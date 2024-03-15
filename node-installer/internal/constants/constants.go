package constants

import (
	_ "embed"
	"path/filepath"

	"github.com/edgelesssys/contrast/node-installer/internal/config"
	"github.com/pelletier/go-toml"
)

var (
	// containerdRuntimeBaseConfig is the configuration file for the containerd runtime
	//
	//go:embed configuration-clh-snp.toml
	containerdRuntimeBaseConfig string

	// containerdBaseConfig is the base configuration file for containerd
	//
	//go:embed containerd-config.toml
	containerdBaseConfig string
)

// CRIFQDN is the fully qualified domain name of the CRI service.
const CRIFQDN = "io.containerd.grpc.v1.cri"

// KataRuntimeConfig returns the Kata runtime configuration.
func KataRuntimeConfig(baseDir string) config.KataRuntimeConfig {
	var config config.KataRuntimeConfig
	if err := toml.Unmarshal([]byte(containerdRuntimeBaseConfig), &config); err != nil {
		panic(err) // should never happen
	}
	config.Hypervisor["clh"]["path"] = filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")
	config.Hypervisor["clh"]["igvm"] = filepath.Join(baseDir, "share", "kata-containers-igvm.img")
	config.Hypervisor["clh"]["image"] = filepath.Join(baseDir, "share", "kata-containers.img")
	config.Hypervisor["clh"]["valid_hypervisor_paths"] = []string{filepath.Join(baseDir, "bin", "cloud-hypervisor-snp")}
	return config
}

// ContainerdBaseConfig returns the base containerd configuration.
func ContainerdBaseConfig() config.ContainerdConfig {
	var config config.ContainerdConfig
	if err := toml.Unmarshal([]byte(containerdBaseConfig), &config); err != nil {
		panic(err) // should never happen
	}
	return config
}

// ContainerdRuntimeConfigFragment returns the containerd runtime configuration fragment.
func ContainerdRuntimeConfigFragment(baseDir string) config.Runtime {
	return config.Runtime{
		Type:           "io.containerd.contrast-cc.v2",
		Path:           filepath.Join(baseDir, "bin", "containerd-shim-contrast-cc-v2"),
		PodAnnotations: []string{"io.katacontainers.*"},
		Options: map[string]any{
			"ConfigPath": filepath.Join(baseDir, "etc", "configuration-clh-snp.toml"),
		},
		PrivilegedWithoutHostDevices: true,
		Snapshotter:                  "tardev",
	}
}

// TardevSnapshotterConfigFragment returns the tardev snapshotter configuration fragment.
func TardevSnapshotterConfigFragment() config.ProxyPlugin {
	return config.ProxyPlugin{
		Type:    "snapshot",
		Address: "/run/containerd/tardev-snapshotter.sock",
	}
}
