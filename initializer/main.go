// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
	"time"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/atls/issuer"
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
		Use:          "initializer",
		Short:        "initializer",
		RunE:         run,
		SilenceUsage: true,
		Version:      constants.Version,
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

	coordinatorHostname := os.Getenv("COORDINATOR_HOST")
	if coordinatorHostname == "" {
		return errors.New("COORDINATOR_HOST not set")
	}

	ctx := cmd.Context()

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
		dial := dialer.NewWithKey(issuer, atls.NoValidators, atls.NoMetrics, &net.Dialer{}, privKey)
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

	var resp *meshapi.NewMeshCertResponse
	for {
		resp, err = requestCert()
		if err == nil {
			log.Info("Requesting cert", "response", resp)
			break
		}
		log.Warn("Requesting cert", "err", err)
		log.Info("Retrying in 10s")
		time.Sleep(10 * time.Second)
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

	log.Info("Initializer done")
	return nil
}
