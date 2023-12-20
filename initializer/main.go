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
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/intercom"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (retErr error) {
	logger := slog.Default()
	defer func() {
		if retErr != nil {
			logger.Error(retErr.Error())
		}
	}()

	logger.Info("Initializer started")

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
	logger.Info("Deriving public key", "pubKeyHash", pubKeyHashStr)

	requestCert := func() (*intercom.NewMeshCertResponse, error) {
		dial := dialer.NewWithKey(snp.NewIssuer(logger), atls.NoValidator, &net.Dialer{}, privKey)
		conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorHostname, intercom.Port))
		if err != nil {
			return nil, fmt.Errorf("dialing: %w", err)
		}
		defer conn.Close()

		client := intercom.NewIntercomClient(conn)

		req := &intercom.NewMeshCertRequest{
			PeerPublicKeyHash: pubKeyHashStr,
		}
		resp, err := client.NewMeshCert(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("calling NewMeshCert: %w", err)
		}
		return resp, nil
	}

	resp := &intercom.NewMeshCertResponse{}
	for {
		resp, err = requestCert()
		if err == nil {
			logger.Info("Requesting cert", "response", resp)
			break
		}
		logger.Warn("Requesting cert", "err", err)
		logger.Info("Retrying in 10s")
		time.Sleep(10 * time.Second)
	}

	// convert privKey to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("marshaling private key: %v", err)
	}
	pemEncodedPrivKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// write files to disk
	err = os.WriteFile("/tls-config/CACert.pem", resp.CaCert, 0o644)
	if err != nil {
		return fmt.Errorf("writing cert.pem: %v", err)
	}
	err = os.WriteFile("/tls-config/certChain.pem", resp.CertChain, 0o644)
	if err != nil {
		return fmt.Errorf("writing cert.pem: %v", err)
	}
	err = os.WriteFile("/tls-config/key.pem", pemEncodedPrivKey, 0o600)
	if err != nil {
		return fmt.Errorf("writing key.pem: %v", err)
	}

	logger.Info("Initializer done")
	return nil
}
