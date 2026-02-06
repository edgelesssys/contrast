// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package targetconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/spf13/afero"
)

// TargetConfig holds the configuration for the target system where the node-installer is running.
type TargetConfig struct {
	containerdConfigPath string
	restartSystemdUnit   bool
	systemdUnitNames     []string
	kataConfigPath       string

	hostMount string
	fs        *afero.Afero
}

// NewTargetConfig creates a targetConfig with default values.
func NewTargetConfig(hostMount, runtimeBase string, pl platforms.Platform) (*TargetConfig, error) {
	conf := &TargetConfig{
		containerdConfigPath: "etc/containerd/config.toml",
		restartSystemdUnit:   true,
		systemdUnitNames:     []string{"containerd.service"},
		hostMount:            hostMount,
		fs:                   &afero.Afero{Fs: afero.NewOsFs()},
	}
	switch {
	case platforms.IsQEMU(pl) && platforms.IsSNP(pl):
		conf.kataConfigPath = filepath.Join(runtimeBase, "etc", "configuration-qemu-snp.toml")
	case platforms.IsQEMU(pl) && platforms.IsTDX(pl):
		conf.kataConfigPath = filepath.Join(runtimeBase, "etc", "configuration-qemu-tdx.toml")
	default:
		return nil, fmt.Errorf("unsupported platform %q", pl)
	}
	return conf, nil
}

// LoadOverridesFromDir loads target configuration overrides from the specified directory.
func (c *TargetConfig) LoadOverridesFromDir(
	targetConfigDir string,
) error {
	if _, err := c.fs.Stat(targetConfigDir); errors.Is(err, fs.ErrNotExist) {
		log.Printf("Target config directory %q does not exist, using default configuration.\n", targetConfigDir)
		return nil
	} else if err != nil {
		return fmt.Errorf("checking target config directory %q: %w", targetConfigDir, err)
	}
	if err := c.fs.Walk(targetConfigDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := c.fs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %q: %w", path, err)
		}
		dataStr := string(bytes.TrimSpace(data))
		if dataStr == "" {
			return fmt.Errorf("%s cannot be empty", info.Name())
		}
		switch info.Name() {
		case "containerd-config-path":
			c.containerdConfigPath = dataStr
		case "restart-systemd-unit":
			restart, err := strconv.ParseBool(dataStr)
			if err != nil {
				return fmt.Errorf("parsing restart-containerd value %q: %w", dataStr, err)
			}
			c.restartSystemdUnit = restart
		case "systemd-unit-name":
			c.systemdUnitNames = strings.FieldsFunc(dataStr, func(r rune) bool {
				return slices.Contains([]string{",", " ", "\n"}, string(r))
			})
		case "kata-config-path":
			// TODO(burgerdev): this config knob should be replaced with one for the full installation directory.
			c.kataConfigPath = dataStr
		default:
			return fmt.Errorf("unexpected file %q in target config dir %q", info.Name(), targetConfigDir)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walking directory %q: %w", targetConfigDir, err)
	}
	return nil
}

// ContainerdConfigPath returns the path to the containerd configuration file.
func (c *TargetConfig) ContainerdConfigPath() string {
	return filepath.Join(c.hostMount, c.containerdConfigPath)
}

// RestartSystemdUnit returns whether the systemd unit should be restarted.
func (c *TargetConfig) RestartSystemdUnit() bool {
	return c.restartSystemdUnit
}

// SystemdUnitNames returns the names of the systemd units to restart.
func (c *TargetConfig) SystemdUnitNames() []string {
	return c.systemdUnitNames
}

// KataConfigPath returns the path to the Kata Containers configuration file.
func (c *TargetConfig) KataConfigPath() string {
	return filepath.Join(c.hostMount, c.kataConfigPath)
}
