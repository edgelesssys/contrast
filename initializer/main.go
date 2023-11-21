package main

import (
	"fmt"
	"log"

	"github.com/google/go-sev-guest/client"
)

func main() {
	log.Println("Initializer started")

	log.Println("Getting extended report")
	snpGuestDevice, err := client.OpenDevice()
	if err != nil {
		log.Fatalf("opening device: %v", err)
	}
	defer snpGuestDevice.Close()

	reportData := [64]byte{}
	report, err := client.GetReport(snpGuestDevice, reportData)
	if err != nil {
		log.Fatalf("getting extended report: %v", err)
	}

	fmt.Printf("Extended report: %v\n", report)

	log.Println("Initializer done")
}
