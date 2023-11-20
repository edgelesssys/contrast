package main

import (
	"log"
	"net/http"

	"github.com/katexochen/coordinator-kbs/internal/ca"
	"github.com/katexochen/coordinator-kbs/internal/kbs"
)

func main() {
	log.Println("Coordinator started")

	ca, err := ca.New()
	if err != nil {
		log.Fatalf("creating CA: %v", err)
	}

	kbsHandler, err := kbs.NewHandler(ca)
	if err != nil {
		log.Fatalf("creating KBS handler: %v", err)
	}

	err = http.ListenAndServe(":2838", kbsHandler)
	if err != nil {
		log.Fatalf("serving KBS: %v", err)
	}
}
