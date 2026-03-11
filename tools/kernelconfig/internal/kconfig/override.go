// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kconfig

import (
	"bytes"
)

// OverrideConfig allows setting and unsetting options of a kernel config.
func OverrideConfig(baseConfig []byte, isGPU bool) (*Config, error) {
	config, err := Parse(bytes.NewReader(baseConfig))
	if err != nil {
		return nil, err
	}

	// Add some options to enable using the kernel in NixOS. (As NixOS has a hard check on
	// whether all modules required for systemd are present, e.g.)
	config.Set("CONFIG_EFIVAR_FS", "y")
	config.Set("CONFIG_RD_ZSTD", "y")
	config.Set("CONFIG_VFAT_FS", "y")
	config.Unset("CONFIG_EXPERT")
	config.Set("CONFIG_NLS_CODEPAGE_437", "y")
	config.Set("CONFIG_NLS_ISO8859_1", "y")
	config.Set("CONFIG_ATA", "y")
	config.Set("CONFIG_ATA_PIIX", "y")
	config.Set("CONFIG_DMIID", "y")

	if isGPU {
		// Disable module signing to make the build reproducible.
		config.Set("CONFIG_MODULE_SIG", "n")
	} else {
		// Our kernel build is independent of any initrd.
		config.Set("CONFIG_INITRAMFS_SOURCE", `""`)
		// NixOS requires capability of loading kernel modules.
		config.Set("CONFIG_MODULES", "y")
	}

	return config, nil
}
