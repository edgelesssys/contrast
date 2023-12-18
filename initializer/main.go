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
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/edgelesssys/nunki/internal/atls"
	"github.com/edgelesssys/nunki/internal/attestation/snp"
	"github.com/edgelesssys/nunki/internal/grpc/dialer"
	"github.com/edgelesssys/nunki/internal/intercom"
)

func main() {
	log.Println("Initializer started")

	coordinatorHostname := os.Getenv("COORDINATOR_HOST")
	if coordinatorHostname == "" {
		log.Fatalf("COORDINATOR_HOST not set")
	}

	ctx := context.Background()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("generating key: %v", err)
	}

	pubKey, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		log.Fatalf("marshaling public key: %v", err)
	}
	pubKeyHash := sha256.Sum256(pubKey)
	pubKeyHashStr := hex.EncodeToString(pubKeyHash[:])
	log.Printf("pubKeyHash: %v", pubKeyHashStr)

	requestCert := func() (*intercom.NewMeshCertResponse, error) {
		dial := dialer.NewWithKey(snp.NewIssuer(), atls.NoValidator, &net.Dialer{}, privKey)
		conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorHostname, intercom.Port))
		if err != nil {
			return nil, fmt.Errorf("dialing: %v", err)
		}
		defer conn.Close()

		client := intercom.NewIntercomClient(conn)

		req := &intercom.NewMeshCertRequest{
			PeerPublicKeyHash: pubKeyHashStr,
		}
		resp, err := client.NewMeshCert(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("Error: calling NewMeshCert: %v", err)
		}
		return resp, nil
	}

	resp := &intercom.NewMeshCertResponse{}
	for {
		resp, err = requestCert()
		if err == nil {
			log.Printf("Response: %v", resp)
			break
		}
		log.Printf("Error: %v", err)
		log.Println("retrying in 10s")
		time.Sleep(10 * time.Second)
	}

	// convert privKey to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		log.Fatalf("marshaling private key: %v", err)
	}
	pemEncodedPrivKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// write files to disk
	err = os.WriteFile("/tls-config/CACert.pem", resp.CaCert, 0o644)
	if err != nil {
		log.Fatalf("writing cert.pem: %v", err)
	}
	err = os.WriteFile("/tls-config/certChain.pem", resp.CertChain, 0o644)
	if err != nil {
		log.Fatalf("writing cert.pem: %v", err)
	}
	err = os.WriteFile("/tls-config/key.pem", pemEncodedPrivKey, 0o600)
	if err != nil {
		log.Fatalf("writing key.pem: %v", err)
	}

	log.Println("Initializer done")
}
