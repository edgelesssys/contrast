// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
	"github.com/edgelesssys/contrast/internal/defaultdeny"
	"github.com/edgelesssys/contrast/internal/grpc/dialer"
	"github.com/edgelesssys/contrast/internal/logger"
	"github.com/edgelesssys/contrast/internal/meshapi"

	"github.com/edgelesssys/contrast/internal/constants"
	"github.com/spf13/cobra"
)

const (
	// workloadSecretPath is fixed path to the Contrast workload secret.
	workloadSecretPath = "/contrast/secrets/workload-secret-seed"
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	cmd := newRootCmd()
	ctx := context.Background()
	return cmd.ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:              "initializer",
		Short:            "initializer",
		PersistentPreRun: printPreface,
		RunE:             run,
		SilenceUsage:     true,
		Version:          constants.Version,
	}
	root.InitDefaultVersionFlag()
	root.AddCommand(
		NewCryptsetupCmd(),
	)
	return root
}

func run(cmd *cobra.Command, _ []string) (retErr error) {
	log, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}
	defer func() {
		if retErr != nil {
			log.Error(retErr.Error())
		}
	}()

	log.Info("Initializer started")

	// If the service mesh is disabled, we don't have a service mesh sidecar
	// container that installs its iptables rules. Therefore, we remove the
	// rule here.
	if os.Getenv(constants.DisableServiceMeshEnvVar) != "" {
		log.Info(fmt.Sprintf("%s set, removing default deny rule", constants.DisableServiceMeshEnvVar))
		if err := defaultdeny.RemoveDefaultDenyRule(log); err != nil {
			return fmt.Errorf("removing default deny rule: %w", err)
		}
	}

	coordinatorHostname := os.Getenv("COORDINATOR_HOST")
	if coordinatorHostname == "" {
		return errors.New("COORDINATOR_HOST not set")
	}

	ctx := cmd.Context()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	issuer, err := issuer.New(log)
	if err != nil {
		return fmt.Errorf("creating issuer: %w", err)
	}

	requestCert := func() (*meshapi.NewMeshCertResponse, error) {
		// Supply an empty list of validators, as the coordinator does not need to be
		// validated by the initializer.
		dial := dialer.NewWithKey(issuer, atls.NoValidators, atls.NoMetrics, nil, privKey, log)
		conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorHostname, meshapi.Port))
		if err != nil {
			return nil, fmt.Errorf("dialing: %w", err)
		}
		defer conn.Close()

		client := meshapi.NewMeshAPIClient(conn)

		resp, err := client.NewMeshCert(ctx, &meshapi.NewMeshCertRequest{})
		if err != nil {
			return nil, fmt.Errorf("calling NewMeshCert: %w", err)
		}
		return resp, nil
	}

	ticker := time.NewTicker(5 * time.Second)
	var resp *meshapi.NewMeshCertResponse
	for {
		resp, err = requestCert()
		if err == nil {
			log.Info("Requesting cert", "response", resp)
			break
		}
		log.Warn("Requesting cert", "err", err)
		log.Info("Waiting for retry")
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return fmt.Errorf("waiting for retry to request cert: %w", ctx.Err())
		}
	}

	// convert privKey to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}
	pemEncodedPrivKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// make sure directories exist
	if err := os.MkdirAll("/contrast/tls-config", 0o755); err != nil {
		return fmt.Errorf("creating tls-config directory: %w", err)
	}
	if err := os.MkdirAll("/contrast/secrets", 0o755); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}

	// write files to disk
	err = os.WriteFile("/contrast/tls-config/mesh-ca.pem", resp.MeshCACert, 0o444)
	if err != nil {
		return fmt.Errorf("writing mesh-ca.pem: %w", err)
	}
	err = os.WriteFile("/contrast/tls-config/certChain.pem", resp.CertChain, 0o444)
	if err != nil {
		return fmt.Errorf("writing certChain.pem: %w", err)
	}
	err = os.WriteFile("/contrast/tls-config/key.pem", pemEncodedPrivKey, 0o400)
	if err != nil {
		return fmt.Errorf("writing key.pem: %w", err)
	}
	err = os.WriteFile("/contrast/tls-config/coordinator-root-ca.pem", resp.RootCACert, 0o444)
	if err != nil {
		return fmt.Errorf("writing coordinator-root-ca.pem: %w", err)
	}

	if len(resp.WorkloadSecret) > 0 {
		err = os.WriteFile(workloadSecretPath, []byte(hex.EncodeToString(resp.WorkloadSecret)), 0o400)
		if err != nil {
			return fmt.Errorf("writing workload-secret-seed: %w", err)
		}
	}

	cryptsetupDevicePath := os.Getenv("CRYPTSETUP_DEVICE")
	if cryptsetupDevicePath != "" {
		log.Info("Setting up encrypted mount")
		flags := &cryptsetupFlags{
			devicePath:       cryptsetupDevicePath,
			volumeMountPoint: "/state",
		}
		if err := setupEncryptedMount(ctx, log, flags); err != nil {
			return fmt.Errorf("setting up encrypted mount: %w", err)
		}
		log.Info("Encrypted mount setup done")
	}

	if err := os.WriteFile("/done", []byte(""), 0o644); err != nil {
		return fmt.Errorf("creating startup probe done directory:%w", err)
	}
	log.Info("Initializer done")

	if cryptsetupDevicePath != "" {
		// The device mount is created by the initializer and shared with the application
		// container through a common emptyDir. We want to avoid mounting on a sub-path of
		// the emptyDir to make configuration for the application container easier, so we
		// mount the encrypted volume on top of the emptyDir mount point. The Kata agent
		// does not know about this and will unmount that mount point when the initializer
		// exits, which affects the application container, too. We avoid this situation by
		// keeping the initializer running indefinitely.
		log.Info("Idling to keep propagated cryptsetup device mounts alive")
		<-ctx.Done()
	}
	return nil
}

func printPreface(cmd *cobra.Command, _ []string) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Contrast initializer %s\n", constants.Version)
	fmt.Fprintln(cmd.ErrOrStderr(), "Report issues at https://github.com/edgelesssys/contrast/issues")
}
