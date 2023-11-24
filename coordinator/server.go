package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/katexochen/coordinator-kbs/internal/atls"
	"github.com/katexochen/coordinator-kbs/internal/attestation/snp"
	"github.com/katexochen/coordinator-kbs/internal/grpc/atlscredentials"
	"github.com/katexochen/coordinator-kbs/internal/intercom"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type intercomServer struct {
	grpc *grpc.Server

	intercom.UnimplementedIntercomServer
}

func newIntercomServer(meshAuth *meshAuthority) (*intercomServer, error) {
	validator := snp.NewValidator(meshAuth)
	credentials := atlscredentials.New(atls.NoIssuer, []atls.Validator{validator})
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
	)
	s := &intercomServer{grpc: grpcServer}
	intercom.RegisterIntercomServer(s.grpc, s)
	return s, nil
}

func (i *intercomServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	return i.grpc.Serve(lis)
}

func (i *intercomServer) NewMeshCert(ctx context.Context, req *intercom.NewMeshCertRequest) (*intercom.NewMeshCertResponse, error) {
	log.Println("NewMeshCert called")

	return &intercom.NewMeshCertResponse{}, nil
}
