// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/asset"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/config"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/containerdconfig"
	"github.com/edgelesssys/contrast/nodeinstaller/internal/kataconfig"
	"github.com/google/go-sev-guest/abi"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	fmt.Fprintf(os.Stderr, "Contrast node-installer %s\n", constants.Version)
	fmt.Fprintln(os.Stderr, "Report issues at https://github.com/edgelesssys/contrast/issues")

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
		targetPath := strings.ReplaceAll(file.Path, kataconfig.RuntimeNamePlaceholder, runtimeHandlerName)

		fmt.Printf("Fetching %q to %q\n", file.URL, targetPath)

		if err := os.MkdirAll(filepath.Dir(filepath.Join(hostMount, targetPath)), 0o777); err != nil {
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
	if err := os.MkdirAll(kataConfigPath, 0o777); err != nil {
		return fmt.Errorf("creating directory %q: %w", kataConfigPath, err)
	}
	var containerdConfigPath string
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-clh-snp.toml")
		containerdConfigPath = filepath.Join(hostMount, "etc", "containerd", "config.toml")
	case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-snp.toml")
		containerdConfigPath = filepath.Join(hostMount, "etc", "containerd", "config.toml")
	case platforms.MetalQEMUTDX:
		kataConfigPath = filepath.Join(kataConfigPath, "configuration-qemu-tdx.toml")
		containerdConfigPath = filepath.Join(hostMount, "etc", "containerd", "config.toml")
	case platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU:
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

	if err := containerdRuntimeConfig(runtimeBase, kataConfigPath, platform, config.QemuExtraKernelParams, config.DebugRuntime); err != nil {
		return fmt.Errorf("generating kata runtime configuration: %w", err)
	}

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return fmt.Errorf("getting runtime handler name: %w", err)
	}

	if err := patchContainerdConfig(runtimeHandler, runtimeBase, containerdConfigPath, platform, config.DebugRuntime); err != nil {
		return fmt.Errorf("patching containerd configuration: %w", err)
	}

	// If the user opted to not have us restart containerd, we're done here.
	if !shouldRestartContainerd {
		return nil
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP, platforms.MetalQEMUSNP, platforms.MetalQEMUTDX,
		platforms.MetalQEMUSNPGPU:
		return restartHostContainerd(ctx, containerdConfigPath, "containerd.service")
	case platforms.K3sQEMUTDX, platforms.K3sQEMUSNP, platforms.K3sQEMUSNPGPU:
		if hostServiceExists("k3s") {
			return restartHostContainerd(ctx, containerdConfigPath, "k3s.service")
		} else if hostServiceExists("k3s-agent") {
			return restartHostContainerd(ctx, containerdConfigPath, "k3s-agent.service")
		}
		return fmt.Errorf("neither k3s nor k3s-agent service found")
	case platforms.RKE2QEMUTDX:
		if hostServiceExists("rke2-server") {
			return restartHostContainerd(ctx, containerdConfigPath, "rke2-server.service")
		} else if hostServiceExists("rke2-agent") {
			return restartHostContainerd(ctx, containerdConfigPath, "rke2-agent.service")
		}
		return fmt.Errorf("neither rke2-server nor rke2-agent service found")
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

func containerdRuntimeConfig(basePath, configPath string, platform platforms.Platform, qemuExtraKernelParams string, debugRuntime bool) error {
	snpIDBlock, err := kataconfig.SnpIDBlockForPlatform(platform, abi.SevProduct().Name)
	if err != nil {
		return err
	}
	kataRuntimeConfig, err := kataconfig.KataRuntimeConfig(basePath, platform, qemuExtraKernelParams, snpIDBlock, debugRuntime)
	if err != nil {
		return fmt.Errorf("generating kata runtime config: %w", err)
	}
	rawConfig, err := kataRuntimeConfig.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling kata runtime config: %w", err)
	}
	return os.WriteFile(configPath, rawConfig, 0o666)
}

func patchContainerdConfig(runtimeHandler, basePath, configPath string, platform platforms.Platform, debugRuntime bool) error {
	existingRaw, existing, err := parseExistingContainerdConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to parse existing containerd config: %v\n", err)
		fmt.Println("Creating a new containerd base config.")
		existing = containerdconfig.Base()
	}

	if debugRuntime {
		// Enable containerd debug logging.
		existing.Debug.Level = "debug"
	}

	// Ensure section for the snapshotter proxy plugin exists.
	if existing.ProxyPlugins == nil {
		existing.ProxyPlugins = make(map[string]containerdconfig.ProxyPlugin)
	}

	var snapshotterName, socketName string
	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		// Add the snapshotter proxy plugin.
		snapshotterName = fmt.Sprintf("tardev-%s", runtimeHandler)
		socketName = fmt.Sprintf("/run/containerd/tardev-snapshotter-%s.sock", runtimeHandler)
		existing.ProxyPlugins[snapshotterName] = containerdconfig.ProxyPlugin{
			Type:    "snapshot",
			Address: socketName,
		}
	}

	// Add contrast-cc runtime
	runtimes := ensureMapPath(&existing.Plugins, containerdconfig.CRIFQDN(existing.Version), "containerd", "runtimes")
	containerdRuntimeConfig, err := containerdconfig.RuntimeFragment(basePath, snapshotterName, platform)
	if err != nil {
		return fmt.Errorf("generating containerd runtime config: %w", err)
	}
	runtimes[runtimeHandler] = containerdRuntimeConfig

	rawConfig, err := toml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshaling containerd config: %w", err)
	}

	if bytes.Equal(existingRaw, rawConfig) {
		fmt.Println("Containerd config already up-to-date. No changes needed.")
		return nil
	}

	// Backup the existing config.
	if len(existingRaw) != 0 {
		if err := os.WriteFile(fmt.Sprintf("%s.%d.bak", configPath, time.Now().Unix()), existingRaw, 0o666); err != nil {
			return fmt.Errorf("backing up existing config: %w", err)
		}
	}

	fmt.Printf("Patching containerd config at %s\n", configPath)
	tmpFile, err := os.CreateTemp(filepath.Dir(configPath), "containerd-config-*.toml")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	if _, err = tmpFile.Write(rawConfig); err != nil {
		return fmt.Errorf("writing to temporary file: %w", err)
	}
	if err := os.Chmod(tmpFile.Name(), 0o666); err != nil {
		return fmt.Errorf("chmod %q: %w", tmpFile.Name(), err)
	}
	return os.Rename(tmpFile.Name(), configPath)
}

func parseExistingContainerdConfig(path string) ([]byte, containerdconfig.Config, error) {
	// Read the rendered config instead of the template, as we can't parse the template.
	// We then write the rendered config to the template path later.
	renderedPath, isRendered := strings.CutSuffix(path, ".tmpl")
	configData, err := os.ReadFile(renderedPath)
	if err != nil {
		return nil, containerdconfig.Config{}, err
	}

	var cfg containerdconfig.Config
	if err := toml.Unmarshal(configData, &cfg); err != nil {
		return nil, containerdconfig.Config{}, err
	}

	if !isRendered {
		return configData, cfg, nil
	}

	// We return the raw file content so that the caller can decide whether to overwrite. Since
	// they are overwriting the template file and not the rendered file, we need to return the
	// template file here.
	configData, err = os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// The template file will be created by us, pretend that it's empty right now.
		return []byte{}, cfg, nil
	} else if err != nil {
		return nil, containerdconfig.Config{}, fmt.Errorf("reading containerd config template %s: %w", path, err)
	}
	return configData, cfg, nil
}

func restartHostContainerd(ctx context.Context, containerdConfigPath, service string) error {
	// get mtime of the config file
	info, err := os.Stat(containerdConfigPath)
	if err != nil {
		return fmt.Errorf("stat %q: %w", containerdConfigPath, err)
	}
	configMtime := info.ModTime()

	startTime, err := getSystemdServiceRestartTime(ctx, service)
	if err != nil {
		return fmt.Errorf("retrieving service (%s) start time: %w", service, err)
	}

	fmt.Printf("service (%s) start time: %s\n", service, startTime.Format(time.RFC3339Nano))
	fmt.Printf("config mtime:          %s\n", configMtime.Format(time.RFC3339Nano))
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

func getSystemdServiceRestartTime(ctx context.Context, service string) (time.Time, error) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("connecting to system bus: %w", err)
	}
	defer conn.Close()

	property, err := conn.GetUnitPropertyContext(ctx, service, "ActiveEnterTimestamp")
	if err != nil {
		return time.Time{}, fmt.Errorf("getting property ActiveEnterTimestamp: %w", err)
	}

	timestamp, ok := property.Value.Value().(uint64)
	if !ok {
		return time.Time{}, fmt.Errorf("wrong type: %T", property.Value.Value())
	}
	return time.Unix(0, int64(timestamp)*1000), nil
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
