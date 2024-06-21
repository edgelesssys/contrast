// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
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

	"github.com/edgelesssys/contrast/node-installer/flavours"
	"github.com/edgelesssys/contrast/node-installer/internal/asset"
	"github.com/edgelesssys/contrast/node-installer/internal/config"
	"github.com/edgelesssys/contrast/node-installer/internal/constants"
	"github.com/pelletier/go-toml/v2"
)

var shouldRestartContainerd = flag.Bool("restart", true, "Restart containerd after the runtime installation to make the changes effective.")

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: node-installer <flavour>")
		os.Exit(1)
	}
	flag.Parse()

	flavour, err := flavours.FromString(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fetcher := asset.NewDefaultFetcher()
	if err := run(context.Background(), fetcher, flavour); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Installation completed successfully.")
}

func run(ctx context.Context, fetcher assetFetcher, flavour flavours.Flavour) error {
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

	runtimeBase := filepath.Join("/opt", "edgeless", config.RuntimeHandlerName)
	binDir := filepath.Join(hostMount, runtimeBase, "bin")

	// Create directory structure
	if err := os.MkdirAll(binDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating runtime bin directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(hostMount, runtimeBase, "share"), os.ModePerm); err != nil {
		return fmt.Errorf("creating runtime share directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(hostMount, runtimeBase, "etc"), os.ModePerm); err != nil {
		return fmt.Errorf("creating runtime etc directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(hostMount, "etc", "containerd"), os.ModePerm); err != nil {
		return fmt.Errorf("creating /etc/containerd directory: %w", err)
	}

	for _, file := range config.Files {
		fmt.Printf("Fetching %q to %q\n", file.URL, file.Path)
		var fetchErr error
		if file.Integrity == "" {
			_, fetchErr = fetcher.FetchUnchecked(ctx, file.URL, filepath.Join(hostMount, file.Path))
		} else {
			_, fetchErr = fetcher.Fetch(ctx, file.URL, filepath.Join(hostMount, file.Path), file.Integrity)
		}
		if fetchErr != nil {
			return fmt.Errorf("fetching file from %q to %q: %w", file.URL, file.Path, fetchErr)
		}
	}
	items, err := os.ReadDir(binDir)
	if err != nil {
		return fmt.Errorf("reading bin directory %q: %w", binDir, err)
	}

	for _, item := range items {
		if !item.Type().IsRegular() {
			continue
		}
		if err := os.Chmod(filepath.Join(binDir, item.Name()), 0o755); err != nil {
			return fmt.Errorf("chmod %q: %w", item.Name(), err)
		}
	}

	kataConfigPath := filepath.Join(hostMount, runtimeBase, "etc")
	var containerdConfigPath string
	switch flavour {
	case flavours.AKSCLHSNP:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-clh-snp.toml")
		containerdConfigPath = filepath.Join(hostMount, "etc", "containerd", "config.toml")
	case flavours.K3sQEMUTDX:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-tdx.toml")
		containerdConfigPath = filepath.Join(hostMount, "var", "lib", "rancher", "k3s", "agent", "etc", "containerd", "config.toml")
	case flavours.RKE2QEMUTDX:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-tdx.toml")
		containerdConfigPath = filepath.Join(hostMount, "var", "lib", "rancher", "rke2", "agent", "etc", "containerd", "config.toml")
	default:
		return fmt.Errorf("unsupported flavour %q", flavour)
	}

	if err := containerdRuntimeConfig(runtimeBase, kataConfigPath, flavour, config.DebugRuntime); err != nil {
		return fmt.Errorf("generating kata runtime configuration: %w", err)
	}

	switch flavour {
	case flavours.AKSCLHSNP:
		// AKS or any external-containerd based K8s distro: We can just patch the existing containerd config at /etc/containerd/config.toml
		if err := patchContainerdConfig(config.RuntimeHandlerName, runtimeBase, containerdConfigPath, flavour); err != nil {
			return fmt.Errorf("patching containerd configuration: %w", err)
		}
	case flavours.K3sQEMUTDX, flavours.RKE2QEMUTDX:
		// K3s or RKE2: We need to extend the configuration template, which, in it's un-templated form, is non-TOML.
		// Therefore just write the TOML configuration fragment ourselves and append it to the template file.
		// This assumes that the user does not yet have a runtime with the same name configured himself,
		// but as our runtimes are hash-named, this should be a safe assumption.
		if err := patchContainerdConfigTemplate(config.RuntimeHandlerName, runtimeBase, containerdConfigPath, flavour); err != nil {
			return fmt.Errorf("patching containerd configuration: %w", err)
		}
	default:
		return fmt.Errorf("unsupported flavour %q", flavour)
	}

	// If the user opted to not have us restart containerd, we're done here.
	if !*shouldRestartContainerd {
		return nil
	}

	switch flavour {
	case flavours.AKSCLHSNP:
		return restartHostContainerd(containerdConfigPath, "containerd")
	case flavours.K3sQEMUTDX:
		if hostServiceExists("k3s") {
			return restartHostContainerd(containerdConfigPath, "k3s")
		} else if hostServiceExists("k3s-agent") {
			return restartHostContainerd(containerdConfigPath, "k3s-agent")
		} else {
			return fmt.Errorf("neither k3s nor k3s-agent service found")
		}
	case flavours.RKE2QEMUTDX:
		if hostServiceExists("rke2-server") {
			return restartHostContainerd(containerdConfigPath, "rke2-server")
		} else if hostServiceExists("rke2-agent") {
			return restartHostContainerd(containerdConfigPath, "rke2-agent")
		} else {
			return fmt.Errorf("neither rke2-server nor rke2-agent service found")
		}

	default:
		return fmt.Errorf("unsupported flavour %q", flavour)
	}
}

func envWithDefault(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return dflt
	}
	return value
}

func containerdRuntimeConfig(basePath, configPath string, flavour flavours.Flavour, debugRuntime bool) error {
	kataRuntimeConfig, err := constants.KataRuntimeConfig(basePath, flavour, debugRuntime)
	if err != nil {
		return fmt.Errorf("generating kata runtime config: %w", err)
	}
	rawConfig, err := toml.Marshal(kataRuntimeConfig)
	if err != nil {
		return fmt.Errorf("marshaling kata runtime config: %w", err)
	}
	return os.WriteFile(configPath, rawConfig, os.ModePerm)
}

func patchContainerdConfig(runtimeName, basePath, configPath string, flavour flavours.Flavour) error {
	existingRaw, existing, err := parseExistingContainerdConfig(configPath)
	if err != nil {
		existing = constants.ContainerdBaseConfig()
	}

	// Add tardev snapshotter, only required for AKS
	if flavour == flavours.AKSCLHSNP {
		if existing.ProxyPlugins == nil {
			existing.ProxyPlugins = make(map[string]config.ProxyPlugin)
		}
		if _, ok := existing.ProxyPlugins["tardev"]; !ok {
			existing.ProxyPlugins["tardev"] = constants.TardevSnapshotterConfigFragment()
		}
	}

	// Add contrast-cc runtime
	runtimes := ensureMapPath(&existing.Plugins, constants.CRIFQDN, "containerd", "runtimes")
	containerdRuntimeConfig, err := constants.ContainerdRuntimeConfigFragment(basePath, flavour)
	if err != nil {
		return fmt.Errorf("generating containerd runtime config: %w", err)
	}
	runtimes[runtimeName] = containerdRuntimeConfig

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

func patchContainerdConfigTemplate(runtimeName, basePath, configTemplatePath string, flavour flavours.Flavour) error {
	existingConfig, err := os.ReadFile(configTemplatePath)
	if err != nil {
		return fmt.Errorf("reading containerd config template: %w", err)
	}

	// Extend a scratchpad config with the new plugin configuration. (including the new contrast-cc runtime)
	var newConfigFragment config.ContainerdConfig
	runtimes := ensureMapPath(&newConfigFragment.Plugins, constants.CRIFQDN, "containerd", "runtimes")
	containerdRuntimeConfig, err := constants.ContainerdRuntimeConfigFragment(basePath, flavour)
	if err != nil {
		return fmt.Errorf("generating containerd runtime config: %w", err)
	}
	runtimes[runtimeName] = containerdRuntimeConfig

	// We purposely don't marshal the full config, as we only want to append the plugin section.
	rawNewPluginConfig, err := toml.Marshal(newConfigFragment.Plugins)
	if err != nil {
		return fmt.Errorf("marshaling containerd runtime config: %w", err)
	}

	// First append the existing config template by a newline, so that if it ends without a newline,
	// the new config fragment isn't appended to the last line..
	newRawConfig := append(existingConfig, []byte("\n")...)
	// ..then append the new config fragment
	newRawConfig = append(newRawConfig, rawNewPluginConfig...)

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
		return fmt.Errorf("getting service start time: %w %q", err, serviceStartTime)
	}

	// format: ActiveEnterTimestamp=Day YYYY-MM-DD HH:MM:SS UTC
	dayUTC := strings.TrimPrefix(strings.TrimSpace(string(serviceStartTime)), "ActiveEnterTimestamp=")
	startTime, err := time.Parse("Mon 2006-01-02 15:04:05 MST", dayUTC)
	if err != nil {
		return fmt.Errorf("parsing service start time: %w", err)
	}

	fmt.Printf("service (%s) start time: %s\n", service, startTime.Format(time.RFC3339))
	fmt.Printf("config mtime:          %s\n", configMtime.Format(time.RFC3339))
	if startTime.After(configMtime) {
		fmt.Println("service already running with the newest config")
		return nil
	}

	// This command will restart containerd on the host and will take down the installer with it.
	out, err := exec.Command(
		"nsenter", "--target", "1", "--mount", "--",
		"systemctl", "restart", service,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restarting service: %w: %s", err, out)
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
		if current[p] == nil {
			current[p] = make(map[string]any)
		}
		current = current[p].(map[string]any)
	}
	return current
}

type assetFetcher interface {
	Fetch(ctx context.Context, sourceURI, destination, integrity string) (changed bool, retErr error)
	FetchUnchecked(ctx context.Context, sourceURI, destination string) (changed bool, retErr error)
}
