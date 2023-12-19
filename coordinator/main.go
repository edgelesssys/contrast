package main

import (
	"log"
	"net"
	"os"

	"github.com/edgelesssys/nunki/internal/ca"
	"github.com/edgelesssys/nunki/internal/coordapi"
	"github.com/edgelesssys/nunki/internal/intercom"
)

func main() {
	log.Println("Coordinator started")

	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		log.Fatalf("NAMESPACE environment variable not set")
	}

	caInstance, err := ca.New(namespace)
	if err != nil {
		log.Fatalf("failed to create CA: %v", err)
	}

	manifestSetGetter := newManifestSetGetter()

	coordS, err := newCoordAPIServer(manifestSetGetter, caInstance)
	if err != nil {
		log.Fatalf("failed to create coordinator API server: %v", err)
	}

	go func() {
		log.Println("Coordinator API listening")
		if err := coordS.Serve(net.JoinHostPort("0.0.0.0", coordapi.Port)); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	log.Println("Waiting for manifest")
	manifest := manifestSetGetter.GetManifest()
	log.Println("Got manifest")

	meshAuth, err := newMeshAuthority(caInstance, manifest)
	if err != nil {
		log.Fatalf("failed to create mesh authority: %v", err)
	}

	intercomS, err := newIntercomServer(meshAuth, caInstance)
	if err != nil {
		log.Fatalf("failed to create intercom server: %v", err)
	}

	log.Println("Coordinator intercom listening")
	if err := intercomS.Serve(net.JoinHostPort("0.0.0.0", intercom.Port)); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
