// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/edgelesssys/contrast/tools/igvm"
	"github.com/spf13/cobra"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd := newRootCmd()
	return cmd.ExecuteContext(context.Background())
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "igvm",
		Short: "igvm",
	}
	cmd.AddCommand(newModfiyCmd())
	return cmd
}

func newModfiyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify <path to igvm>",
		Short: "modify an existing IGVM image",
		RunE:  runModify,
	}
	cmd.Flags().StringP("output", "o", "", "output file path (default: overwrite input file, use '-' for stdout)")
	must(cmd.MarkFlagFilename("output"))
	cmd.Flags().String("snp-id-block", "", "overwrite SNP IDBlock (path to JSON file)")
	must(cmd.MarkFlagFilename("snp-id-block"))
	return cmd
}

func runModify(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	flags, err := parseModifyFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	igvmFile, err := readIGVM(args[0])
	if err != nil {
		return fmt.Errorf("reading igvm file: %w", err)
	}

	if flags.snpIDBlockPath != "" {
		var idBlockUpdate igvm.VhsSnpIDBlock
		f, err := os.ReadFile(flags.snpIDBlockPath)
		if err != nil {
			return fmt.Errorf("reading file from %q: %w", flags.snpIDBlockPath, err)
		}
		if err := json.Unmarshal(f, &idBlockUpdate); err != nil {
			return fmt.Errorf("unmarshaling snp id block from json: %w", err)
		}
		for i, vhs := range igvmFile.VariableHeaders {
			if vhs.Type == igvm.VhtSnpIDBlock {
				idBlockUpdateBytes, err := idBlockUpdate.MarshalBinary()
				if err != nil {
					return fmt.Errorf("marshaling snp id block to binary: %w", err)
				}
				igvmFile.VariableHeaders[i].Content = idBlockUpdateBytes
				break
			}
		}
	}

	if err := igvmFile.UpdateChecksum(); err != nil {
		return fmt.Errorf("updating checksum: %w", err)
	}

	igvmData, err := igvmFile.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshaling igvm: %w", err)
	}

	if flags.outputPath == "" {
		flags.outputPath = args[0]
	}

	if flags.outputPath == "-" {
		if _, err := os.Stdout.Write(igvmData); err != nil {
			return fmt.Errorf("writing igvm to stdout: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(flags.outputPath, igvmData, 0o644); err != nil {
		return fmt.Errorf("writing igvm to %q: %w", flags.outputPath, err)
	}

	return nil
}

type modifyFlags struct {
	outputPath     string
	snpIDBlockPath string
}

func parseModifyFlags(cmd *cobra.Command) (modifyFlags, error) {
	var flags modifyFlags
	var err error
	flags.outputPath, err = cmd.Flags().GetString("output")
	if err != nil {
		return flags, fmt.Errorf("failed to get output path: %w", err)
	}
	flags.snpIDBlockPath, err = cmd.Flags().GetString("snp-id-block")
	if err != nil {
		return flags, fmt.Errorf("failed to get SNP IDBlock path: %w", err)
	}
	return flags, nil
}

func readIGVM(path string) (*igvm.IGVM, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file from %q: %w", path, err)
	}

	var igvmFile igvm.IGVM
	if err := igvmFile.UnmarshalBinary(f); err != nil {
		return nil, fmt.Errorf("unmarshaling igvm file: %w", err)
	}
	return &igvmFile, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
