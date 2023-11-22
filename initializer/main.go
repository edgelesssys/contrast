package main

import (
	"context"
	"log"
	"net"
	"os"

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

	dial := dialer.New(snp.NewIssuer(), atls.NoVerifier, &net.Dialer{})
	conn, err := dial.Dial(ctx, net.JoinHostPort(coordinatorIP, intercom.Port))
	if err != nil {
		log.Fatalf("dialing: %v", err)
	}
	defer conn.Close()

	client := intercom.NewIntercomClient(conn)

	req := &intercom.NewMeshCertRequest{}
	resp, err := client.NewMeshCert(context.Background(), req)
	if err != nil {
		log.Fatalf("calling NewMeshCert: %v", err)
	}
	log.Printf("Response: %v", resp)

	log.Println("Initializer done")
}
