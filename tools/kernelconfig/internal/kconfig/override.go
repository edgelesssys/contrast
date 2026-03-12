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
		config.Set("CONFIG_MD", "y")
		config.Set("CONFIG_MODULE_SIG", "n")

		// Enable dm-init, so that we can use `dm-mod.create`.
		// Source: https://github.com/kata-containers/kata-containers/blob/2c6126d3ab708e480b5aad1e7f7adbe22ffaa539/tools/packaging/kernel/configs/fragments/common/confidential_containers/cryptsetup.conf
		config.Set("CONFIG_BLK_DEV_DM", "y")
		config.Set("CONFIG_DM_INIT", "y")
		config.Set("CONFIG_DM_CRYPT", "y")
		config.Set("CONFIG_DM_VERITY", "y")
		config.Set("CONFIG_DM_INTEGRITY", "y")
	} else {
		// Our kernel build is independent of any initrd.
		config.Set("CONFIG_INITRAMFS_SOURCE", `""`)
		// Enable dm-init, so that we can use `dm-mod.create`.
		config.Set("CONFIG_DM_INIT", "y")
		// NixOS requires capability of loading kernel modules.
		config.Set("CONFIG_MODULES", "y")
	}

	return config, nil
}
