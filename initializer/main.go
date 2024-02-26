package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/logger"
	"github.com/edgelesssys/nunki/internal/meshapi"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (retErr error) {
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

	ctx := context.Background()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	pubKey, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}
	pubKeyHash := sha256.Sum256(pubKey)
	pubKeyHashStr := hex.EncodeToString(pubKeyHash[:])
	log.Info("Deriving public key", "pubKeyHash", pubKeyHashStr)

	requestCert := func() (*meshapi.NewMeshCertResponse, error) {
		issuer := snp.NewIssuer(logger.NewNamed(log, "snp-issuer"))
		dial := dialer.NewWithKey(issuer, atls.NoValidator, &net.Dialer{}, privKey)
		conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorHostname, meshapi.Port))
		if err != nil {
			return nil, fmt.Errorf("dialing: %w", err)
		}
		defer conn.Close()

		client := meshapi.NewMeshAPIClient(conn)

		req := &meshapi.NewMeshCertRequest{
			PeerPublicKeyHash: pubKeyHashStr,
		}
		resp, err := client.NewMeshCert(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("calling NewMeshCert: %w", err)
		}
		return resp, nil
	}

	resp := &meshapi.NewMeshCertResponse{}
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

	// write files to disk
	err = os.WriteFile("/tls-config/MeshCACert.pem", resp.MeshCACert, 0o644)
	if err != nil {
		return fmt.Errorf("writing MeshCACert.pem: %w", err)
	}
	err = os.WriteFile("/tls-config/certChain.pem", resp.CertChain, 0o644)
	if err != nil {
		return fmt.Errorf("writing certChain.pem: %w", err)
	}
	err = os.WriteFile("/tls-config/key.pem", pemEncodedPrivKey, 0o600)
	if err != nil {
		return fmt.Errorf("writing key.pem: %w", err)
	}
	err = os.WriteFile("/tls-config/RootCACert.pem", resp.RootCACert, 0o644)
	if err != nil {
		return fmt.Errorf("writing RootCACert.pem: %w", err)
	}

	log.Info("Initializer done")
	return nil
}
