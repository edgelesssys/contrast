package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/katexochen/coordinator-kbs/internal/intercom"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Coordinator started")

	manifestEnv := os.Getenv("MANIFEST")
	if manifestEnv == "" {
		log.Fatalf("MANIFEST not set")
	}

	manifestStr, err := base64.StdEncoding.DecodeString(manifestEnv)
	if err != nil {
		log.Fatalf("decoding manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestStr, &manifest); err != nil {
		log.Fatalf("unmarshaling manifest: %v", err)
	}

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
