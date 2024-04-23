// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/edgelesssys/contrast/cli/cmd"
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

var version = "0.0.0-dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
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
		cmd.NewRuntimeCmd(),
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
