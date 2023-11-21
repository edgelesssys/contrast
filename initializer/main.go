package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/google/go-sev-guest/client"
	"github.com/katexochen/coordinator-kbs/internal/intercom"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Initializer started")

	coordinatorIP := os.Getenv("COORDINATOR_IP")
	if coordinatorIP == "" {
		log.Fatalf("COORDINATOR_IP not set")
	}

	log.Println("Getting extended report")
	snpGuestDevice, err := client.OpenDevice()
	if err != nil {
		log.Fatalf("opening device: %v", err)
	}
	defer snpGuestDevice.Close()

	reportData := [64]byte{}
	report, err := client.GetRawReport(snpGuestDevice, reportData)
	if err != nil {
		log.Fatalf("getting extended report: %v", err)
	}

	fmt.Printf("Extended report: %v\n", report)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(net.JoinHostPort(coordinatorIP, intercom.Port), grpcOpts...)
	if err != nil {
		log.Fatalf("dialing coordinator: %v", err)
	}
	defer conn.Close()

	client := intercom.NewIntercomClient(conn)

	req := &intercom.NewMeshCertRequest{
		AttestationReport: base64.StdEncoding.EncodeToString(report),
	}
	resp, err := client.NewMeshCert(context.Background(), req)
	if err != nil {
		log.Fatalf("calling NewMeshCert: %v", err)
	}
	log.Printf("Response: %v", resp)

	log.Println("Initializer done")
}
