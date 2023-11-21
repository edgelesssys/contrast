package main

import (
	"context"
	"log"

	"github.com/katexochen/coordinator-kbs/internal/intercom"
)

type intercomServer struct {
	intercom.UnimplementedIntercomServer
}

func (i *intercomServer) NewMeshCert(ctx context.Context, req *intercom.NewMeshCertRequest) (*intercom.NewMeshCertResponse, error) {
	log.Println("NewMeshCert called")
	log.Printf("Request: %v", req)
	return &intercom.NewMeshCertResponse{}, nil
}
