package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/node-installer/internal/asset"
	"github.com/edgelesssys/contrast/node-installer/internal/config"
	"github.com/edgelesssys/contrast/node-installer/internal/constants"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	fetcher := asset.NewDefaultFetcher()
	if err := run(context.Background(), fetcher); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Installation completed successfully.")
}

func run(ctx context.Context, fetcher assetFetcher) error {
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
	clhConfigPath := filepath.Join(hostMount, runtimeBase, "etc", "configuration-clh-snp.toml")
	if err := containerdRuntimeConfig(runtimeBase, clhConfigPath); err != nil {
		return fmt.Errorf("generating clh_config.toml: %w", err)
	}
	containerdConfigPath := filepath.Join(hostMount, "etc", "containerd", "config.toml")
	if err := patchContainerdConfig(config.RuntimeHandlerName, runtimeBase, containerdConfigPath); err != nil {
		return fmt.Errorf("patching containerd config: %w", err)
	}

	return restartHostContainerd(containerdConfigPath)
}

func envWithDefault(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return dflt
	}
	return value
}

func containerdRuntimeConfig(basePath, configPath string) error {
	kataRuntimeConfig := constants.KataRuntimeConfig(basePath)
	rawConfig, err := toml.Marshal(kataRuntimeConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, rawConfig, os.ModePerm)
}

func patchContainerdConfig(runtimeName, basePath, configPath string) error {
	existingRaw, existing, err := parseExistingContainerdConfig(configPath)
	if err != nil {
		existing = constants.ContainerdBaseConfig()
	}

	// Add tardev snapshotter
	if existing.ProxyPlugins == nil {
		existing.ProxyPlugins = make(map[string]config.ProxyPlugin)
	}
	if _, ok := existing.ProxyPlugins["tardev"]; !ok {
		existing.ProxyPlugins["tardev"] = constants.TardevSnapshotterConfigFragment()
	}

	// Add contrast-cc runtime
	runtimes := ensureMapPath(&existing.Plugins, constants.CRIFQDN, "containerd", "runtimes")
	runtimes[runtimeName] = constants.ContainerdRuntimeConfigFragment(basePath)

	rawConfig, err := toml.Marshal(existing)
	if err != nil {
		return err
	}

	if slices.Equal(existingRaw, rawConfig) {
		fmt.Println("Containerd config already up-to-date. No changes needed.")
		return nil
	}

	fmt.Println("Patching containerd config")
	return os.WriteFile(configPath, rawConfig, os.ModePerm)
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

func restartHostContainerd(containerdConfigPath string) error {
	// get mtime of the config file
	info, err := os.Stat(containerdConfigPath)
	if err != nil {
		return fmt.Errorf("stat %q: %w", containerdConfigPath, err)
	}
	configMtime := info.ModTime()

	// get containerd start time
	// Note that "--timestamp=unix" is not supported in the installed version of systemd (v250) at the time of writing.
	containerdStartTime, err := exec.Command(
		"nsenter", "--target", "1", "--mount", "--",
		"systemctl", "show", "--timestamp=utc", "--property=ActiveEnterTimestamp", "containerd",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("getting containerd start time: %w %q", err, containerdStartTime)
	}

	// format: ActiveEnterTimestamp=Day YYYY-MM-DD HH:MM:SS UTC
	dayUTC := strings.TrimPrefix(strings.TrimSpace(string(containerdStartTime)), "ActiveEnterTimestamp=")
	startTime, err := time.Parse("Mon 2006-01-02 15:04:05 MST", dayUTC)
	if err != nil {
		return fmt.Errorf("parsing containerd start time: %w", err)
	}

	fmt.Printf("containerd start time: %s\n", startTime.Format(time.RFC3339))
	fmt.Printf("config mtime:          %s\n", configMtime.Format(time.RFC3339))
	if startTime.After(configMtime) {
		fmt.Println("containerd already running with the newest config")
		return nil
	}

	// This command will restart containerd on the host and will take down the installer with it.
	out, err := exec.Command(
		"nsenter", "--target", "1", "--mount", "--",
		"systemctl", "restart", "containerd",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restarting containerd: %w: %s", err, out)
	}
	fmt.Printf("containerd restarted: %s\n", out)
	return nil
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
