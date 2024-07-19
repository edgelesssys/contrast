// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"

	"github.com/edgelesssys/contrast/cli/cmd"
	"github.com/edgelesssys/contrast/cli/constants"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/spf13/cobra"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd := newRootCmd()
	ctx, cancel := signalContext(context.Background(), os.Interrupt)
	defer cancel()
	return cmd.ExecuteContext(ctx)
}

func buildVersionString() string {
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
	if refValues, err := json.MarshalIndent(manifest.EmbeddedReferenceValues(), "\t", "  "); err == nil {
		fmt.Fprintf(versionsWriter, "embedded reference values:\t%s\n", refValues)
	}
	fmt.Fprintf(versionsWriter, "genpolicy version:\t%s\n", constants.GenpolicyVersion)
	versionsWriter.Flush()
	return versionsBuilder.String()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:              "contrast",
		Short:            "contrast",
		PersistentPreRun: preRunRoot,
		Version:          buildVersionString(),
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

	return root
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
