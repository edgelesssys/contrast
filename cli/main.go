package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

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
	cmd := &cobra.Command{
		Short:            "nunki",
		PersistentPreRun: preRunRoot,
		Version:          version,
	}
	cmd.SetOut(os.Stdout)

	cmd.PersistentFlags().String("log-level", "warn", "set logging level (debug, info, warn, error, or a number)")

	cmd.InitDefaultVersionFlag()
	cmd.AddCommand(
		newGenerateCmd(),
		newSetCmd(),
		newVerifyCmd(),
	)

	return cmd
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

func must(err error) {
	if err != nil {
		panic(err)
	}
}
