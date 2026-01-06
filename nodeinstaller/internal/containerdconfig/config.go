// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1
package containerdconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
)

// Config is a containerd config file.
type Config struct {
	path   string
	raw    []byte
	config config
}

// FromPath reads the containerd config from the given path.
func FromPath(path string) (*Config, error) {
	// Read the rendered config instead of the template, as we can't parse the template.
	// We then write the rendered config to the template path later.
	renderedPath, isRendered := strings.CutSuffix(path, ".tmpl")
	configData, err := os.ReadFile(renderedPath)
	if err != nil {
		return nil, err
	}

	var cfg config
	if err := toml.Unmarshal(configData, &cfg); err != nil {
		return nil, err
	}

	if !isRendered {
		return &Config{raw: configData, config: cfg, path: path}, nil
	}

	// Save the original byte content so we can decide later whether to overwrite.
	// Since we will overwrite the template file and not the rendered file,
	// we need to return the template file content here in case it is a template.
	configData, err = os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// The template file will be created by us, pretend that it's empty right now.
		return &Config{raw: nil, config: cfg, path: path}, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading containerd config template %s: %w", path, err)
	}
	return &Config{raw: configData, config: cfg, path: path}, nil
}

// AddRuntime adds a runtime to the containerd config.
func (c *Config) AddRuntime(handler string, runtime Runtime) {
	runtimes := ensureMapPath(&c.config.Plugins, criFQDN(c.config.Version), "containerd", "runtimes")
	runtimes[handler] = runtime
}

// EnableDebug enables debug logging in the containerd config.
func (c *Config) EnableDebug() {
	c.config.Debug.Level = "debug"
}

// Write writes the containerd config back to disk.
// It will create a backup of the existing config.
func (c *Config) Write() error {
	rawConfig, err := toml.Marshal(c.config)
	if err != nil {
		return fmt.Errorf("marshaling containerd config: %w", err)
	}

	if bytes.Equal(c.raw, rawConfig) {
		log.Println("Containerd config already up-to-date. No changes needed.")
		return nil
	}

	if len(c.raw) != 0 {
		t := time.Now().Unix()
		if err := os.WriteFile(fmt.Sprintf("%s.%d.bak", c.path, t), c.raw, 0o666); err != nil {
			return fmt.Errorf("backing up existing config: %w", err)
		}
		log.Printf("Created backup of existing containerd config at %s.%d.bak\n", c.path, t)
	}

	log.Printf("Patching containerd config at %s\n", c.path)
	tmpFile, err := os.CreateTemp(filepath.Dir(c.path), "containerd-config-*.toml")
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
	if err := os.Rename(tmpFile.Name(), c.path); err != nil {
		return fmt.Errorf("renaming temporary file to %q: %w", c.path, err)
	}

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
		cur, ok := current[p].(map[string]any)
		if !ok || cur == nil {
			cur = make(map[string]any)
			current[p] = cur
		}
		current = cur
	}
	return current
}

// ContrastRuntime returns the containerd runtime configuration fragment.
func ContrastRuntime(baseDir string, platform platforms.Platform) (Runtime, error) {
	cfg := Runtime{
		Type:                         "io.containerd.contrast-cc.v2",
		Path:                         filepath.Join(baseDir, "bin", "containerd-shim-contrast-cc-v2"),
		PodAnnotations:               []string{"io.katacontainers.*"},
		PrivilegedWithoutHostDevices: true,
	}

	switch platform {
	case platforms.MetalQEMUTDX, platforms.MetalQEMUTDXGPU:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-tdx.toml"),
		}
	case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-snp.toml"),
		}
	default:
		return Runtime{}, fmt.Errorf("unsupported platform: %s", platform)
	}

	// For GPU support, we need to pass through the CDI annotations.
	if platforms.IsGPU(platform) {
		cfg.PodAnnotations = append(cfg.PodAnnotations, "cdi.k8s.io/*")
	}

	return cfg, nil
}

// criFQDN is the fully qualified domain name of the CRI service, which depends on the containerd config version.
func criFQDN(v int) string {
	switch v {
	case 3:
		return "io.containerd.cri.v1.runtime"
	default:
		return "io.containerd.grpc.v1.cri"
	}
}

// Config provides containerd configuration data.
// This is a simplified version of the actual struct.
// Source: https://github.com/containerd/containerd/blob/dcf2847247e18caba8dce86522029642f60fe96b/services/server/config/config.go#L35
type config struct {
	// Version of the config file
	Version int `toml:"version"`
	// Root is the path to a directory where containerd will store persistent data
	Root string `toml:"root,omitempty"`
	// State is the path to a directory where containerd will store transient data
	State string `toml:"state,omitempty"`
	// TempDir is the path to a directory where to place containerd temporary files
	TempDir string `toml:"temp,omitempty"`
	// PluginDir is the directory for dynamic plugins to be stored
	PluginDir string `toml:"plugin_dir,omitempty"`
	// GRPC configuration settings
	GRPC any `toml:"grpc,omitempty"`
	// TTRPC configuration settings
	TTRPC any `toml:"ttrpc,omitempty"`
	// Debug and profiling settings
	Debug debug `toml:"debug,omitempty"`
	// Metrics and monitoring settings
	Metrics any `toml:"metrics,omitempty"`
	// DisabledPlugins are IDs of plugins to disable. Disabled plugins won't be
	// initialized and started.
	DisabledPlugins []string `toml:"disabled_plugins,omitempty"`
	// RequiredPlugins are IDs of required plugins. Containerd exits if any
	// required plugin doesn't exist or fails to be initialized or started.
	RequiredPlugins []string `toml:"required_plugins,omitempty"`
	// Plugins provides plugin specific configuration for the initialization of a plugin
	Plugins map[string]any `toml:"plugins,omitempty"`
	// OOMScore adjust the containerd's oom score
	OOMScore int `toml:"oom_score,omitempty"`
	// Cgroup specifies cgroup information for the containerd daemon process
	Cgroup any `toml:"cgroup,omitempty"`
	// ProxyPlugins configures plugins which are communicated to over GRPC
	ProxyPlugins map[string]proxyPlugin `toml:"proxy_plugins,omitempty"`
	// Timeouts specified as a duration
	Timeouts map[string]string `toml:"timeouts,omitempty"`
	// Imports are additional file path list to config files that can overwrite main config file fields
	Imports []string `toml:"imports,omitempty"`
	// StreamProcessors configuration
	StreamProcessors map[string]any `toml:"stream_processors,omitempty"`
}

// proxyPlugin provides a proxy plugin configuration.
type proxyPlugin struct {
	Type    string `toml:"type"`
	Address string `toml:"address"`
}

// Runtime defines a containerd Runtime.
type Runtime struct {
	// Type is the runtime type to use in containerd e.g. io.containerd.runtime.v1.linux
	Type string `toml:"runtime_type" json:"runtimeType"`
	// Path is an optional field that can be used to overwrite path to a shim runtime binary.
	// When specified, containerd will ignore runtime name field when resolving shim location.
	// Path must be abs.
	Path string `toml:"runtime_path,omitempty" json:"runtimePath,omitempty"`
	// PodAnnotations is a list of pod annotations passed to both pod sandbox as well as
	// container OCI annotations.
	PodAnnotations []string `toml:"pod_annotations" json:"PodAnnotations"`
	// ContainerAnnotations is a list of container annotations passed through to the OCI config of the containers.
	// Container annotations in CRI are usually generated by other Kubernetes node components (i.e., not users).
	// Currently, only device plugins populate the annotations.
	ContainerAnnotations []string `toml:"container_annotations,omitempty" json:"ContainerAnnotations,omitempty"`
	// Options are config options for the runtime.
	Options map[string]any `toml:"options,omitempty" json:"options,omitempty"`
	// PrivilegedWithoutHostDevices overloads the default behaviour for adding host devices to the
	// runtime spec when the container is privileged. Defaults to false.
	PrivilegedWithoutHostDevices bool `toml:"privileged_without_host_devices,omitempty" json:"privileged_without_host_devices,omitempty"`
	// PrivilegedWithoutHostDevicesAllDevicesAllowed overloads the default behaviour device allowlisting when
	// to the runtime spec when the container when PrivilegedWithoutHostDevices is already enabled. Requires
	// PrivilegedWithoutHostDevices to be enabled. Defaults to false.
	PrivilegedWithoutHostDevicesAllDevicesAllowed bool `toml:"privileged_without_host_devices_all_devices_allowed,omitempty" json:"privileged_without_host_devices_all_devices_allowed,omitempty"`
	// BaseRuntimeSpec is a json file with OCI spec to use as base spec that all container's will be created from.
	BaseRuntimeSpec string `toml:"base_runtime_spec,omitempty" json:"baseRuntimeSpec,omitempty"`
	// NetworkPluginConfDir is a directory containing the CNI network information for the runtime class.
	NetworkPluginConfDir string `toml:"cni_conf_dir,omitempty" json:"cniConfDir,omitempty"`
	// NetworkPluginMaxConfNum is the max number of plugin config files that will
	// be loaded from the cni config directory by go-cni. Set the value to 0 to
	// load all config files (no arbitrary limit). The legacy default value is 1.
	NetworkPluginMaxConfNum int `toml:"cni_max_conf_num,omitempty" json:"cniMaxConfNum,omitempty"`
	// Snapshotter setting snapshotter at runtime level instead of making it as a global configuration.
	// An example use case is to use devmapper or other snapshotters in Kata containers for performance and security
	// while using default snapshotters for operational simplicity.
	// See https://github.com/containerd/containerd/issues/6657 for details.
	Snapshotter string `toml:"snapshotter,omitempty" json:"snapshotter,omitempty"`
	// Sandboxer defines which sandbox runtime to use when scheduling pods
	// This features requires the new CRI server implementation (enabled by default in 2.0)
	// shim - means use whatever Controller implementation provided by shim (e.g. use RemoteController).
	// podsandbox - means use Controller implementation from sbserver podsandbox package.
	Sandboxer string `toml:"sandboxer,omitempty" json:"sandboxer,omitempty"`
}

// debug provides debug configuration.
type debug struct {
	Address string `toml:"address,omitempty"`
	UID     int    `toml:"uid,omitempty"`
	GID     int    `toml:"gid,omitempty"`
	Level   string `toml:"level,omitempty"`
	// Format represents the logging format. Supported values are 'text' and 'json'.
	Format string `toml:"format,omitempty"`
}
