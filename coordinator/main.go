package main

import (
	"log"
	"net"

	"github.com/katexochen/coordinator-kbs/internal/intercom"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Coordinator started")

	lis, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", intercom.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	intercom.RegisterIntercomServer(s, &intercomServer{})

	log.Println("Coordinator listening")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
