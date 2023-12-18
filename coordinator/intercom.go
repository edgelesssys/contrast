package main

import (
	"context"
	"fmt"
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
	grpc          *grpc.Server
	certGet       certGetter
	caChainGetter certChainGetter

	intercom.UnimplementedIntercomServer
}

type certGetter interface {
	GetCert(peerPublicKeyHashStr string) ([]byte, error)
}

func newIntercomServer(meshAuth *meshAuthority, caGetter certChainGetter) (*intercomServer, error) {
	validator := snp.NewValidatorWithCallbacks(meshAuth, meshAuth)
	credentials := atlscredentials.New(atls.NoIssuer, []atls.Validator{validator})
	grpcServer := grpc.NewServer(
		grpc.Creds(credentials),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 15 * time.Second}),
	)
	s := &intercomServer{
		grpc:          grpcServer,
		certGet:       meshAuth,
		caChainGetter: caGetter,
	}
	intercom.RegisterIntercomServer(s.grpc, s)
	return s, nil
}

func (i *intercomServer) Serve(endpoint string) error {
	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	return i.grpc.Serve(lis)
}

func (i *intercomServer) NewMeshCert(ctx context.Context, req *intercom.NewMeshCertRequest,
) (*intercom.NewMeshCertResponse, error) {
	log.Println("NewMeshCert called")

	cert, err := i.certGet.GetCert(req.PeerPublicKeyHash)
	if err != nil {
		return nil, err
	}

	intermCert := i.caChainGetter.GetIntermCert()

	return &intercom.NewMeshCertResponse{
		// TODO(3u13r): Replace the CA Cert the intermediate CA cert
		CaCert:    i.caChainGetter.GetCACert(),
		CertChain: append(intermCert, cert...),
	}, nil
}
