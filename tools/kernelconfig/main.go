// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/edgelesssys/contrast/tools/kernelconfig/internal/base"
	"github.com/edgelesssys/contrast/tools/kernelconfig/internal/kconfig"
)

func main() {
	gpu := flag.Bool("gpu", false, "Use GPU configuration")
	flag.Parse()

	baseConfig := base.BaseConfig
	if *gpu {
		baseConfig = base.BaseConfigGPU
	}

	cfg, err := kconfig.OverrideConfig(baseConfig, *gpu)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, arg := range flag.Args() {
		key, value, ok := strings.Cut(arg, "=")
		if !ok || !strings.HasPrefix(key, "CONFIG_") || value == "" {
			fmt.Fprintf(os.Stderr, "Error: invalid or empty argument %q (expected CONFIG_KEY=VALUE)\n", arg)
			os.Exit(1)
		}
		if value == "unset" {
			cfg.Unset(key)
		} else {
			cfg.Set(key, value)
		}
	}

	fmt.Print(string(cfg.Marshal()))
}
