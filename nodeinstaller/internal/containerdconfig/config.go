// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1
package containerdconfig

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
)

// base is the base configuration file for containerd
//
//go:embed containerd-config.toml
var base string

// Base returns the base containerd configuration.
func Base() Config {
	var config Config
	if err := toml.Unmarshal([]byte(base), &config); err != nil {
		panic(err) // should never happen
	}
	return config
}

// RuntimeFragment returns the containerd runtime configuration fragment.
func RuntimeFragment(baseDir, snapshotter string, platform platforms.Platform) (*Runtime, error) {
	cfg := Runtime{
		Type:                         "io.containerd.contrast-cc.v2",
		Path:                         filepath.Join(baseDir, "bin", "containerd-shim-contrast-cc-v2"),
		PodAnnotations:               []string{"io.katacontainers.*"},
		PrivilegedWithoutHostDevices: true,
	}

	switch platform {
	case platforms.AKSCloudHypervisorSNP:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-clh-snp.toml"),
		}
		cfg.Snapshotter = snapshotter
	case platforms.MetalQEMUTDX:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-tdx.toml"),
		}
	case platforms.MetalQEMUSNP, platforms.MetalQEMUSNPGPU:
		cfg.Options = map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-qemu-snp.toml"),
		}
		// For GPU support, we need to pass through the CDI annotations.
		if platforms.IsGPU(platform) {
			cfg.PodAnnotations = append(cfg.PodAnnotations, "cdi.k8s.io/*")
		}
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return &cfg, nil
}

// CRIFQDN is the fully qualified domain name of the CRI service, which depends on the containerd config version.
func CRIFQDN(v int) string {
	switch v {
	case 3:
		return "io.containerd.cri.v1.runtime"
	default:
		return "io.containerd.grpc.v1.cri"
	}
}

// ImagesFQDN is the fully qualified domain name of the images plugin, which was factored out of the CRI plugin in containerd v2.
func ImagesFQDN(v int) string {
	switch v {
	case 3:
		return "io.containerd.cri.v1.images"
	default:
		return "io.containerd.grpc.v1.cri"
	}
}

// Config provides containerd configuration data.
// This is a simplified version of the actual struct.
// Source: https://github.com/containerd/containerd/blob/dcf2847247e18caba8dce86522029642f60fe96b/services/server/config/config.go#L35
type Config struct {
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
	Debug Debug `toml:"debug,omitempty"`
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
	ProxyPlugins map[string]ProxyPlugin `toml:"proxy_plugins,omitempty"`
	// Timeouts specified as a duration
	Timeouts map[string]string `toml:"timeouts,omitempty"`
	// Imports are additional file path list to config files that can overwrite main config file fields
	Imports []string `toml:"imports,omitempty"`
	// StreamProcessors configuration
	StreamProcessors map[string]any `toml:"stream_processors,omitempty"`
}

// ProxyPlugin provides a proxy plugin configuration.
type ProxyPlugin struct {
	Type    string `toml:"type"`
	Address string `toml:"address"`
}

// Runtime defines a containerd runtime.
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

// Debug provides debug configuration.
type Debug struct {
	Address string `toml:"address,omitempty"`
	UID     int    `toml:"uid,omitempty"`
	GID     int    `toml:"gid,omitempty"`
	Level   string `toml:"level,omitempty"`
	// Format represents the logging format. Supported values are 'text' and 'json'.
	Format string `toml:"format,omitempty"`
}
