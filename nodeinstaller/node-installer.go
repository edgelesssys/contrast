// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/asset"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/config"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/constants"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	shouldRestartContainerd := flag.Bool("restart", true, "Restart containerd after the runtime installation to make the changes effective.")
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("Usage: node-installer <platform>")
		os.Exit(1)
	}

	platform, err := platforms.FromString(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fetcher := asset.NewDefaultFetcher()
	if err := run(context.Background(), fetcher, platform, *shouldRestartContainerd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Installation completed successfully.")
}

func run(ctx context.Context, fetcher assetFetcher, platform platforms.Platform, shouldRestartContainerd bool) error {
	configDir := envWithDefault("CONFIG_DIR", "/config")
	hostMount := envWithDefault("HOST_MOUNT", "/host")

	configPath := filepath.Join(configDir, "contrast-node-install.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config %q: %w", configPath, err)
	}
	var config config.Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("parsing config %q: %w", configPath, err)
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	fmt.Printf("Using config: %+v\n", config)

	runtimeHandlerName, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return fmt.Errorf("getting runtime handler name: %w", err)
	}

	// Copy the files
	for _, file := range config.Files {
		// Replace @@runtimeName@@ in the target path with the actual base directory.
		targetPath := strings.ReplaceAll(file.Path, constants.RuntimeNamePlaceholder, runtimeHandlerName)

		fmt.Printf("Fetching %q to %q\n", file.URL, targetPath)

		if err := os.MkdirAll(filepath.Dir(filepath.Join(hostMount, targetPath)), os.ModePerm); err != nil {
			return fmt.Errorf("creating directory %q: %w", filepath.Dir(targetPath), err)
		}

		var fetchErr error
		if file.Integrity == "" {
			_, fetchErr = fetcher.FetchUnchecked(ctx, file.URL, filepath.Join(hostMount, targetPath))
		} else {
			_, fetchErr = fetcher.Fetch(ctx, file.URL, filepath.Join(hostMount, targetPath), file.Integrity)
		}
		if fetchErr != nil {
			return fmt.Errorf("fetching file from %q to %q: %w", file.URL, targetPath, fetchErr)
		}

		if file.Executable {
			if err := os.Chmod(filepath.Join(hostMount, targetPath), 0o755); err != nil {
				return fmt.Errorf("chmod %q: %w", targetPath, err)
			}
		}
	}

	runtimeBase := filepath.Join("/opt", "edgeless", runtimeHandlerName)
	kataConfigPath := filepath.Join(hostMount, runtimeBase, "etc")
	if err := os.MkdirAll(kataConfigPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating directory %q: %w", kataConfigPath, err)
	}
	var containerdConfigPath string
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-clh-snp.toml")
		containerdConfigPath = filepath.Join(hostMount, "etc", "containerd", "config.toml")
	case platforms.K3sQEMUSNP:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-snp.toml")
		containerdConfigPath = filepath.Join(hostMount, "var", "lib", "rancher", "k3s", "agent", "etc", "containerd", "config.toml.tmpl")
	case platforms.K3sQEMUTDX:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-tdx.toml")
		containerdConfigPath = filepath.Join(hostMount, "var", "lib", "rancher", "k3s", "agent", "etc", "containerd", "config.toml.tmpl")
	case platforms.RKE2QEMUTDX:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-tdx.toml")
		containerdConfigPath = filepath.Join(hostMount, "var", "lib", "rancher", "rke2", "agent", "etc", "containerd", "config.toml.tmpl")
	default:
		return fmt.Errorf("unsupported platform %q", platform)
	}

	if err := containerdRuntimeConfig(runtimeBase, kataConfigPath, platform, config.DebugRuntime); err != nil {
		return fmt.Errorf("generating kata runtime configuration: %w", err)
	}

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return fmt.Errorf("getting runtime handler name: %w", err)
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		// AKS or any external-containerd based K8s distro: We can just patch the existing containerd config at /etc/containerd/config.toml
		if err := patchContainerdConfig(runtimeHandler, runtimeBase, containerdConfigPath, platform); err != nil {
			return fmt.Errorf("patching containerd configuration: %w", err)
		}
	case platforms.K3sQEMUTDX, platforms.K3sQEMUSNP, platforms.RKE2QEMUTDX:
		// K3s or RKE2: We need to extend the configuration template, which, in it's un-templated form, is non-TOML.
		// Therefore just write the TOML configuration fragment ourselves and append it to the template file.
		// This assumes that the user does not yet have a runtime with the same name configured himself,
		// but as our runtimes are hash-named, this should be a safe assumption.
		if err := patchContainerdConfigTemplate(runtimeHandler, runtimeBase, containerdConfigPath, platform); err != nil {
			return fmt.Errorf("patching containerd configuration: %w", err)
		}
	default:
		return fmt.Errorf("unsupported platform %q", platform)
	}

	// If the user opted to not have us restart containerd, we're done here.
	if !shouldRestartContainerd {
		return nil
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		return restartHostContainerd(containerdConfigPath, "containerd")
	case platforms.K3sQEMUTDX, platforms.K3sQEMUSNP:
		if hostServiceExists("k3s") {
			return restartHostContainerd(containerdConfigPath, "k3s")
		} else if hostServiceExists("k3s-agent") {
			return restartHostContainerd(containerdConfigPath, "k3s-agent")
		} else {
			return fmt.Errorf("neither k3s nor k3s-agent service found")
		}
	case platforms.RKE2QEMUTDX:
		if hostServiceExists("rke2-server") {
			return restartHostContainerd(containerdConfigPath, "rke2-server")
		} else if hostServiceExists("rke2-agent") {
			return restartHostContainerd(containerdConfigPath, "rke2-agent")
		} else {
			return fmt.Errorf("neither rke2-server nor rke2-agent service found")
		}

	default:
		return fmt.Errorf("unsupported platform %q", platform)
	}
}

func envWithDefault(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return dflt
	}
	return value
}

func containerdRuntimeConfig(basePath, configPath string, platform platforms.Platform, debugRuntime bool) error {
	kataRuntimeConfig, err := constants.KataRuntimeConfig(basePath, platform, debugRuntime)
	if err != nil {
		return fmt.Errorf("generating kata runtime config: %w", err)
	}
	rawConfig, err := toml.Marshal(kataRuntimeConfig)
	if err != nil {
		return fmt.Errorf("marshaling kata runtime config: %w", err)
	}
	return os.WriteFile(configPath, rawConfig, os.ModePerm)
}

func patchContainerdConfig(runtimeHandler, basePath, configPath string, platform platforms.Platform) error {
	existingRaw, existing, err := parseExistingContainerdConfig(configPath)
	if err != nil {
		existing = constants.ContainerdBaseConfig()
	}

	snapshotterName := "no-snapshotter"
	// Add tardev snapshotter, only required for AKS
	if platform == platforms.AKSCloudHypervisorSNP {
		if existing.ProxyPlugins == nil {
			existing.ProxyPlugins = make(map[string]config.ProxyPlugin)
		}
		snapshotterName = fmt.Sprintf("tardev-%s", runtimeHandler)
		socketName := fmt.Sprintf("/run/containerd/tardev-snapshotter-%s.sock", runtimeHandler)
		existing.ProxyPlugins[snapshotterName] = config.ProxyPlugin{
			Type:    "snapshot",
			Address: socketName,
		}
	}

	// Add contrast-cc runtime
	runtimes := ensureMapPath(&existing.Plugins, constants.CRIFQDN, "containerd", "runtimes")
	containerdRuntimeConfig, err := constants.ContainerdRuntimeConfigFragment(basePath, snapshotterName, platform)
	if err != nil {
		return fmt.Errorf("generating containerd runtime config: %w", err)
	}
	runtimes[runtimeHandler] = containerdRuntimeConfig

	rawConfig, err := toml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshaling containerd config: %w", err)
	}

	if slices.Equal(existingRaw, rawConfig) {
		fmt.Println("Containerd config already up-to-date. No changes needed.")
		return nil
	}

	fmt.Println("Patching containerd config")
	return os.WriteFile(configPath, rawConfig, os.ModePerm)
}

func patchContainerdConfigTemplate(runtimeHandler, basePath, configTemplatePath string, platform platforms.Platform) error {
	existingConfig, err := os.ReadFile(configTemplatePath)
	if err != nil {
		return fmt.Errorf("reading containerd config template: %w", err)
	}
	fmt.Printf("Existing containerd config template:\n%s\n", existingConfig)

	// Don't add the runtime section if it already exists.
	runtimeSection := fmt.Sprintf("[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.%s]", runtimeHandler)
	if bytes.Contains(existingConfig, []byte(runtimeSection)) {
		fmt.Printf("Runtime section %q already exists\n", runtimeSection)
		return nil
	}

	// PluginFragment contains just the `Plugins` property used to configure containerd.
	type PluginFragment struct {
		// Plugins provides plugin specific configuration for the initialization of a plugin
		Plugins map[string]any `toml:"plugins"`
		// ProxyPlugins provides a map of proxy plugins to be used by containerd
		ProxyPlugins map[string]config.ProxyPlugin `toml:"proxy_plugins"`
	}

	// Extend a scratchpad config with the new plugin configuration. (including the new contrast-cc runtime)
	var newConfigFragment PluginFragment
	newConfigFragment.ProxyPlugins = make(map[string]config.ProxyPlugin)
	snapshotterName := fmt.Sprintf("nydus-%s", runtimeName)
	socketName := fmt.Sprintf("/run/containerd/containerd-nydus-grpc-%s.sock", runtimeName)
	newConfigFragment.ProxyPlugins[snapshotterName] = config.ProxyPlugin{
		Type:    "snapshot",
		Address: socketName,
	}
	runtimes := ensureMapPath(&newConfigFragment.Plugins, constants.CRIFQDN, "containerd", "runtimes")
	containerdRuntimeConfig, err := constants.ContainerdRuntimeConfigFragment(basePath, snapshotterName, platform)
	if err != nil {
		return fmt.Errorf("generating containerd runtime config: %w", err)
	}
	runtimes[runtimeHandler] = containerdRuntimeConfig

	rawNewPluginConfig, err := toml.Marshal(newConfigFragment)
	if err != nil {
		return fmt.Errorf("marshaling containerd runtime config: %w", err)
	}

	// First append the existing config template by a newline, so that if it ends without a newline,
	// the new config fragment isn't appended to the last line..
	newRawConfig := append(existingConfig, []byte("\n")...)

	// The marshalled config is shaped like a tree with the important bits at
	// the leaves and lots of empty parent nodes (except for the link the one
	// child):
	// A > B > C > D > E=foo
	//             | > F > G=bar
	//                 | > H=baz
	// Nodes that don't contain any keys, but only contain a links to children
	// are marshalled as empty sections:
	// ```toml
	// [A] # <- empty section
	// [A.B] # <- empty section
	// [A.B.C] # <- empty section
	// [A.B.C.D] # <- non-empty section
	// E = "foo"
	// [A.B.C.D.F] # <- non-empty section
	// G = "bar"
	// H = "baz"
	// ```
	// We want to avoid appending empty sections (i.e. sections with only a
	// section header, but no keys) to the file because there's a chance that
	// they already exist in the template and creating duplicate sections is
	// illegal.
	// On the other hand, omitting intermediate empty sections is always legal
	// in TOML, so there's no risk in omitting empty sections.
	//
	// We iterate over the marshalled config line by line. If we encounter a
	// section header, we don't immediately append it to the file, but buffer
	// it in `pendingHeaderSection`. If the next line is also a section header,
	// we discard the value in `pendingHeaderSection` and fill it with the new
	// section header. If we encounter a non-section header line, we flush the
	// value of `pendingHeaderSection` to the file, before adding the
	// non-section header line.
	//
	// TODO(freax13): One day, go-toml might add an option to omit empty
	//                sections: https://github.com/pelletier/go-toml/issues/957
	var pendingHeaderSection []byte
	scanner := bufio.NewScanner(bytes.NewReader(rawNewPluginConfig))
	for scanner.Scan() {
		// Is the line a section header?
		if strings.HasPrefix(scanner.Text(), "[") {
			pendingHeaderSection = scanner.Bytes()
			continue
		}
		if len(pendingHeaderSection) != 0 {
			newRawConfig = append(newRawConfig, pendingHeaderSection...)
			newRawConfig = append(newRawConfig, []byte("\n")...)
			pendingHeaderSection = []byte{}
		}
		newRawConfig = append(newRawConfig, scanner.Bytes()...)
		newRawConfig = append(newRawConfig, []byte("\n")...)
	}

	fmt.Printf("New containerd config template:\n%s\n", newRawConfig)
	return os.WriteFile(configTemplatePath, newRawConfig, os.ModePerm)
}

func parseExistingContainerdConfig(path string) ([]byte, config.ContainerdConfig, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, config.ContainerdConfig{}, err
	}

	var cfg config.ContainerdConfig
	if err := toml.Unmarshal(configData, &cfg); err != nil {
		return nil, config.ContainerdConfig{}, err
	}

	return configData, cfg, nil
}

func restartHostContainerd(containerdConfigPath, service string) error {
	// get mtime of the config file
	info, err := os.Stat(containerdConfigPath)
	if err != nil {
		return fmt.Errorf("stat %q: %w", containerdConfigPath, err)
	}
	configMtime := info.ModTime()

	// get containerd start time
	// Note that "--timestamp=unix" is not supported in the installed version of systemd (v250) at the time of writing.
	serviceStartTime, err := exec.Command(
		"nsenter", "--target", "1", "--mount", "--",
		"systemctl", "show", "--timestamp=utc", "--property=ActiveEnterTimestamp", service,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("getting service (%s) start time: %w %q", service, err, serviceStartTime)
	}

	// format: ActiveEnterTimestamp=Day YYYY-MM-DD HH:MM:SS UTC
	dayUTC := strings.TrimPrefix(strings.TrimSpace(string(serviceStartTime)), "ActiveEnterTimestamp=")
	startTime, err := time.Parse("Mon 2006-01-02 15:04:05 MST", dayUTC)
	if err != nil {
		return fmt.Errorf("parsing service (%s) start time: %w", service, err)
	}

	fmt.Printf("service (%s) start time: %s\n", service, startTime.Format(time.RFC3339))
	fmt.Printf("config mtime:          %s\n", configMtime.Format(time.RFC3339))
	if startTime.After(configMtime) {
		fmt.Printf("service (%s) already running with the newest config\n", service)
		return nil
	}

	// This command will restart containerd on the host and will take down the installer with it.
	out, err := exec.Command(
		"nsenter", "--target", "1", "--mount", "--",
		"systemctl", "restart", service,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restarting service (%s): %w: %s", service, err, out)
	}
	fmt.Printf("service (%s) restarted: %s\n", service, out)
	return nil
}

func hostServiceExists(service string) bool {
	if err := exec.Command("nsenter", "--target", "1", "--mount", "--",
		"systemctl", "status", service).Run(); err != nil {
		return false
	}
	return true
}

// ensureMapPath ensures that the given path exists in the map and
// returns the last map in the chain.
func ensureMapPath(in *map[string]any, path ...string) map[string]any {
	if len(path) == 0 {
		return *in
	}
	if *in == nil {
		*in = make(map[string]any)
	}
	current := *in
	for _, p := range path {
		cur, ok := current[p].(map[string]any)
		if !ok || cur == nil {
			cur = make(map[string]any)
			current[p] = cur
		}
		current = cur
	}
	return current
}

type assetFetcher interface {
	Fetch(ctx context.Context, sourceURI, destination, integrity string) (changed bool, retErr error)
	FetchUnchecked(ctx context.Context, sourceURI, destination string) (changed bool, retErr error)
}
