package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"log"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/verify"
	"github.com/katexochen/coordinator-kbs/internal/intercom"
)

type intercomServer struct {
	intercom.UnimplementedIntercomServer
}

func (i *intercomServer) NewMeshCert(ctx context.Context, req *intercom.NewMeshCertRequest) (*intercom.NewMeshCertResponse, error) {
	log.Println("NewMeshCert called")

	reportBytes, err := base64.StdEncoding.DecodeString(req.AttestationReport)
	if err != nil {
		log.Fatalf("decoding attestation report: %v", err)
	}

	report, err := abi.ReportToProto(reportBytes)
	if err != nil {
		log.Fatalf("converting report to proto: %v", err)
	}
	log.Printf("Report: %v", report)

	if err := verify.SnpReport(report, &verify.Options{}); err != nil {
		log.Fatalf("verifying report: %v", err)
	}
	log.Println("Report verified")

	log.Printf("HOSTDATA: %v", hex.EncodeToString(report.HostData))

	return &intercom.NewMeshCertResponse{}, nil
}
