package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/katexochen/coordinator-kbs/internal/atls"
	"github.com/katexochen/coordinator-kbs/internal/attestation/snp"
	"github.com/katexochen/coordinator-kbs/internal/grpc/dialer"
	"github.com/katexochen/coordinator-kbs/internal/intercom"
)

func main() {
	log.Println("Initializer started")

	coordinatorIP := os.Getenv("COORDINATOR_IP")
	if coordinatorIP == "" {
		log.Fatalf("COORDINATOR_IP not set")
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
		conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorIP, intercom.Port))
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

	for {
		resp, err := requestCert()
		if err == nil {
			log.Printf("Response: %v", resp)
			break
		}
		log.Printf("Error: %v", err)
		log.Println("retrying in 10s")
		time.Sleep(10 * time.Second)
	}

	log.Println("Initializer done")
}
