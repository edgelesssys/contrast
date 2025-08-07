// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/spf13/cobra"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd, err := newRootCmd()
	if err != nil {
		return fmt.Errorf("create root cmd: %w", err)
	}

	ctx, cancel := signalContext(context.Background(), os.Interrupt)
	defer cancel()
	return cmd.ExecuteContext(ctx)
}

func buildVersionString() (string, error) {
	var versionsBuilder strings.Builder
	versionsWriter := tabwriter.NewWriter(&versionsBuilder, 0, 0, 4, ' ', 0)
	fmt.Fprintf(versionsWriter, "%s\n\n", constants.Version)

	fmt.Fprintf(versionsWriter, "container image versions:\n")
	imageReplacements := strings.Trim(string(cmd.ReleaseImageReplacements), "\n")
	for _, image := range strings.Split(imageReplacements, "\n") {
		if !strings.HasPrefix(image, "#") {
			image = strings.Split(image, "=")[1]
			fmt.Fprintf(versionsWriter, "\t%s\n", image)
		}
	}

	embeddedReferenceValues, err := manifest.GetEmbeddedReferenceValues()
	if err != nil {
		return "", fmt.Errorf("getting embedded reference values: %w", err)
	}
	for _, platform := range platforms.All() {
		fmt.Fprintf(versionsWriter, "\nreference values for %s platform:\n", platform.String())

		runtimeHandlerName, err := manifest.RuntimeHandler(platform)
		if err != nil {
			return "", fmt.Errorf("getting runtime handler name: %w", err)
		}
		fmt.Fprintf(versionsWriter, "\truntime handler:\t%s\n", runtimeHandlerName)

		values, err := embeddedReferenceValues.ForPlatform(platform)
		if err != nil {
			return "", fmt.Errorf("getting reference values: %w", err)
		}
		printOptionalSVN := func(label string, value *manifest.SVN) {
			fmt.Fprintf(versionsWriter, "\t      %s:\t", label)
			if value != nil {
				fmt.Fprintf(versionsWriter, "%d", value.UInt8())
			} else {
				fmt.Fprint(versionsWriter, "(no default)")
			}
			fmt.Fprint(versionsWriter, "\n")
		}
		for _, snp := range values.SNP {
			fmt.Fprintf(versionsWriter, "\t- launch digest:\t%s\n", snp.TrustedMeasurement.String())
			fmt.Fprint(versionsWriter, "\t  default SNP TCB:\t\n")
			printOptionalSVN("bootloader", snp.MinimumTCB.BootloaderVersion)
			printOptionalSVN("tee", snp.MinimumTCB.TEEVersion)
			printOptionalSVN("snp", snp.MinimumTCB.SNPVersion)
			printOptionalSVN("microcode", snp.MinimumTCB.MicrocodeVersion)
		}
		for _, tdx := range values.TDX {
			fmt.Fprintf(versionsWriter, "\t- mrTd:\t%s\n", tdx.MrTd.String())
			for i, rtmr := range tdx.Rtrms {
				fmt.Fprintf(versionsWriter, "\t  rtrm[%d]:\t%s\n", i, rtmr.String())
			}
			fmt.Fprintf(versionsWriter, "\t  mrSeam:\t%s\n", tdx.MrSeam.String())
			fmt.Fprintf(versionsWriter, "\t  tdAttributes:\t%s\n", tdx.TdAttributes.String())
			fmt.Fprintf(versionsWriter, "\t  xfam:\t%s\n", tdx.Xfam.String())
		}

		switch platform {
		case platforms.AKSCloudHypervisorSNP:
			fmt.Fprintf(versionsWriter, "\tgenpolicy version:\t%s\n", constants.MicrosoftGenpolicyVersion)
		case platforms.MetalQEMUSNP, platforms.MetalQEMUTDX, platforms.MetalQEMUSNPGPU:
			fmt.Fprintf(versionsWriter, "\tgenpolicy version:\t%s\n", constants.KataGenpolicyVersion)
		}
	}

	versionsWriter.Flush()
	return versionsBuilder.String(), nil
}

func newRootCmd() (*cobra.Command, error) {
	version, err := buildVersionString()
	if err != nil {
		return nil, fmt.Errorf("build version string: %w", err)
	}
	root := &cobra.Command{
		Use:              "contrast",
		Short:            "contrast",
		PersistentPreRun: preRunRoot,
		Version:          version,
	}
	root.SetOut(os.Stdout)

	root.PersistentFlags().String("log-level", "warn", "set logging level (debug, info, warn, error, or a number)")
	root.PersistentFlags().String("workspace-dir", "", "directory to write files to, if not set explicitly to another location")

	root.InitDefaultVersionFlag()
	root.AddCommand(
		cmd.NewGenerateCmd(),
		cmd.NewSetCmd(),
		cmd.NewVerifyCmd(),
		cmd.NewRecoverCmd(),
	)

	return root, nil
}

// signalContext returns a context that is canceled on the handed signal.
// The signal isn't watched after its first occurrence. Call the cancel
// function to ensure the internal goroutine is stopped and the signal isn't
// watched any longer.
func signalContext(ctx context.Context, sig os.Signal) (context.Context, context.CancelFunc) {
	sigCtx, stop := signal.NotifyContext(ctx, sig)
	done := make(chan struct{}, 1)
	stopDone := make(chan struct{}, 1)

	go func() {
		defer func() { stopDone <- struct{}{} }()
		defer stop()
		select {
		case <-sigCtx.Done():
			fmt.Println("\rSignal caught. Press ctrl+c again to terminate the program immediately.")
		case <-done:
		}
	}()

	cancelFunc := func() {
		done <- struct{}{}
		<-stopDone
	}

	return sigCtx, cancelFunc
}

func preRunRoot(cmd *cobra.Command, _ []string) {
	cmd.SilenceUsage = true
}
