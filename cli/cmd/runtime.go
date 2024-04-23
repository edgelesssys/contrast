// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"github.com/spf13/cobra"
)

// This value is injected at build time.
var runtimeHandler = "contrast-cc"

// NewRuntimeCmd creates the contrast runtime subcommand.
func NewRuntimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Prints the runtimeClassName",
		Long: `Prints runtimeClassName used by Contrast.

Contrast uses a custom container runtime, where every pod is a confidential
virtual machine. Pod specs of workloads running on Contrast must
have the runtimeClassName set to the value returned by this command.
		`,
		Run: runRuntime,
	}

	return cmd
}

func runRuntime(cmd *cobra.Command, _ []string) {
	cmd.Println(runtimeHandler)
}
