package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/katexochen/coordinator-kbs/internal/intercom"
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
	var manifest *Manifest
	if err := json.Unmarshal(manifestStr, &manifest); err != nil {
		log.Fatalf("unmarshaling manifest: %v", err)
	}

	meshAuth, err := newMeshAuthority(manifest)
	if err != nil {
		log.Fatalf("failed to create mesh authority: %v", err)
	}

	s, err := newIntercomServer(meshAuth)
	if err != nil {
		log.Fatalf("failed to create intercom server: %v", err)
	}

	log.Println("Coordinator listening")
	if err := s.Serve(net.JoinHostPort("0.0.0.0", intercom.Port)); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
