// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	"github.com/edgelesssys/contrast/nodeinstaller/internal/targetconfig"
	"github.com/google/go-sev-guest/abi"
)

func main() {
	fmt.Fprintf(os.Stderr, "Contrast node-installer %s\n", constants.Version)
	fmt.Fprintln(os.Stderr, "Report issues at https://github.com/edgelesssys/contrast/issues")

	if len(os.Args) < 2 {
		log.Fatal("Usage: node-installer <platform>")
	}

	platform, err := platforms.FromString(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fetcher := asset.NewDefaultFetcher()
	if err := run(context.Background(), fetcher, platform); err != nil {
		log.Fatal(err)
	}
	log.Println("Installation completed successfully.")
}

func run(ctx context.Context, fetcher assetFetcher, platform platforms.Platform) error {
	configDir := envWithDefault("CONFIG_DIR", "/config")
	targetConfigDir := envWithDefault("TARGET_CONFIG_DIR", "/target-config")
	hostMount := envWithDefault("HOST_MOUNT", "/host")

	// node-installer configuration, baked into the image.
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
	log.Printf("Using config: %+v\n", config)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		return fmt.Errorf("getting runtime handler name: %w", err)
	}
	runtimeBase := filepath.Join("/opt", "edgeless", runtimeHandler)

	// target config, which can be overridden by the user via configMap.
	targetConf, err := targetconfig.NewTargetConfig(hostMount, runtimeBase, platform)
	if err != nil {
		return fmt.Errorf("creating target config: %w", err)
	}
	if err := targetConf.LoadOverridesFromDir(targetConfigDir); err != nil {
		return fmt.Errorf("loading target config from %q: %w", targetConfigDir, err)
	}
	log.Printf("Using target config: %+v\n", targetConf)

	if err := installFiles(ctx, fetcher, &config, hostMount, runtimeHandler); err != nil {
		return fmt.Errorf("installing files: %w", err)
	}

	imagepullerConfigPath, err := installImagepullerConfig(hostMount, runtimeHandler)
	if err != nil {
		return fmt.Errorf("installing imagepuller config: %w", err)
	}

	if err := containerdRuntimeConfig(runtimeBase, targetConf.KataConfigPath(), platform, config.QemuExtraKernelParams, imagepullerConfigPath, config.DebugRuntime); err != nil {
		return fmt.Errorf("generating kata runtime configuration: %w", err)
	}

	containerdConf, err := containerdconfig.FromPath(targetConf.ContainerdConfigPath())
	if err != nil {
		return fmt.Errorf("loading containerd configuration from %q: %w", targetConf.ContainerdConfigPath(), err)
	}
	if config.DebugRuntime {
		// This only ever happens in development builds of the node-installer.
		containerdConf.EnableDebug()
	}
	containerdRuntime, err := containerdconfig.ContrastRuntime(runtimeBase, platform)
	if err != nil {
		return fmt.Errorf("generating containerd runtime fragment: %w", err)
	}
	containerdConf.AddRuntime(runtimeHandler, containerdRuntime)
	if err := containerdConf.Write(); err != nil {
		return fmt.Errorf("writing containerd config %q: %w", targetConf.ContainerdConfigPath(), err)
	}

	if targetConf.RestartSystemdUnit() {
		if err := restartHostContainerd(ctx, targetConf.ContainerdConfigPath(), targetConf.SystemdUnitNames()); err != nil {
			return fmt.Errorf("restarting systemd unit: %w", err)
		}
	}

	return nil
}

// installFiles fetches and installs files specified in the node-installer configuration file
// from the node-installer container image to the host filesystem.
func installFiles(
	ctx context.Context,
	fetcher assetFetcher,
	config *config.Config,
	hostMount string,
	runtimeHandlerName string,
) error {
	for _, file := range config.Files {
		// Replace @@runtimeName@@ in the target path with the actual base directory.
		targetPath := strings.ReplaceAll(file.Path, kataconfig.RuntimeNamePlaceholder, runtimeHandlerName)

		log.Printf("Fetching %q to %q\n", file.URL, targetPath)

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
	return nil
}

func envWithDefault(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return dflt
	}
	return value
}

func containerdRuntimeConfig(basePath, configPath string, platform platforms.Platform, qemuExtraKernelParams, imagepullerConfigPath string, debugRuntime bool) error {
	var snpIDBlock kataconfig.SnpIDBlock
	if platforms.IsSNP(platform) && platforms.IsQEMU(platform) {
		var err error
		snpIDBlock, err = kataconfig.SnpIDBlockForPlatform(platform, abi.SevProduct().Name)
		if err != nil {
			return fmt.Errorf("getting SNP ID block for platform %q: %w", platform, err)
		}
	}
	kataRuntimeConfig, err := kataconfig.KataRuntimeConfig(basePath, platform, qemuExtraKernelParams, snpIDBlock, imagepullerConfigPath, debugRuntime)
	if err != nil {
		return fmt.Errorf("generating kata runtime config: %w", err)
	}
	rawConfig, err := kataRuntimeConfig.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling kata runtime config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", filepath.Dir(configPath), err)
	}
	if err := os.WriteFile(configPath, rawConfig, 0o666); err != nil {
		return fmt.Errorf("writing kata runtime config to %q: %w", configPath, err)
	}
	return nil
}

func restartHostContainerd(ctx context.Context, containerdConfigPath string, serviceNames []string) error {
	// Go through list of possible service names and check if one exists.
	service := ""
	for _, s := range serviceNames {
		if hostServiceExists(ctx, s) {
			service = s
			break
		}
	}
	if service == "" {
		return fmt.Errorf("no systemd service with name in %v found", serviceNames)
	}

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

	log.Printf("Service (%s) start time: %s\n", service, startTime.Format(time.RFC3339Nano))
	log.Printf("Containerd config mtime:          %s\n", configMtime.Format(time.RFC3339Nano))
	if startTime.After(configMtime) {
		log.Printf("Service (%s) already running with the newest config\n", service)
		return nil
	}

	// This command will restart containerd on the host and will take down the installer with it.
	out, err := exec.CommandContext(
		ctx, "nsenter", "--target", "1", "--mount", "--",
		"systemctl", "restart", service,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restarting service (%s): %w: %s", service, err, out)
	}
	log.Printf("Service (%s) restarted\n", service)
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

func hostServiceExists(ctx context.Context, service string) bool {
	if err := exec.CommandContext(ctx, "nsenter", "--target", "1", "--mount", "--",
		"systemctl", "status", service).Run(); err != nil {
		return false
	}
	return true
}

type assetFetcher interface {
	Fetch(ctx context.Context, sourceURI, destination, integrity string) (changed bool, retErr error)
	FetchUnchecked(ctx context.Context, sourceURI, destination string) (changed bool, retErr error)
}

// installImagepullerConfig installs the (optional) imagepuller config file onto the host filesystem.
func installImagepullerConfig(
	hostMount string,
	runtimeHandlerName string,
) (string, error) {
	imagepullerSource := "/imagepuller-config/contrast-imagepuller.toml"
	configPath := fmt.Sprintf("/opt/edgeless/%s/etc/host-config/contrast-imagepuller.toml", runtimeHandlerName)
	configPathHost := filepath.Join(hostMount, configPath)

	if _, err := os.Stat(imagepullerSource); err != nil {
		log.Println("No imagepuller config provided, skipping")
		return "", nil
	}

	if err := os.MkdirAll(filepath.Dir(configPathHost), 0o777); err != nil {
		return "", fmt.Errorf("creating imagepuller source configuration parent dir %q: %w", filepath.Dir(configPathHost), err)
	}

	inputFile, err := os.Open(imagepullerSource)
	if err != nil {
		return "", fmt.Errorf("opening imagepuller config source file %q: %w", imagepullerSource, err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(configPathHost)
	if err != nil {
		return "", fmt.Errorf("opening imagepuller config destination file %q: %w", configPathHost, err)
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return "", fmt.Errorf("copying imagepuller config from source %q to destination %q: %w", imagepullerSource, configPathHost, err)
	}

	log.Printf("Installed imagepuller config to %q on the host", configPath)
	return configPath, nil
}
