package main

import (
	"bytes"
	"context"
	"io"
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

	return cmd
}

func run() {
	f, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	if len(f) < 0x16 {
		panic("file too small")
	}

	var igvmFile igvm.IGVM
	if err := igvmFile.BinaryUnmarshal(f); err != nil {
		panic(err)
	}

	var idblock igvm.VhsSnpIDBlock
	for i, vhs := range igvmFile.VariableHeaders {
		if vhs.Type == igvm.VhtSnpIdBlock {
			if err := idblock.BinaryUnmarshal(vhs.Content); err != nil {
				panic(err)
			}
			idblockBytes, err := idblock.BinaryMarshal()
			if err != nil {
				panic(err)
			}
			igvmFile.VariableHeaders[i].Content = idblockBytes
			break
		}
	}

	igvmData, err := igvmFile.BinaryMarshal()
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, bytes.NewBuffer(igvmData))
}
