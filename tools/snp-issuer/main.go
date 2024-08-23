package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/client"
)

func main() {
	log.Println("Started...")

	for {
		var reportData [64]byte
		copy(reportData[:], "fuuu")

		log.Println("Getting quote provider...")
		quoteProvider := &client.LinuxIoctlQuoteProvider{}
		if !quoteProvider.IsSupported() {
			panic("ioctl quote provider is not supported")
		}

		log.Println("Getting raw quote...")
		reportRaw, err := quoteProvider.GetRawQuote(reportData)
		if err != nil {
			panic(fmt.Errorf("getting raw report: %w", err))
		}
		log.Println("Report:", hex.EncodeToString(reportRaw))

		log.Println("Parsing report...")
		if _, err := abi.ReportToProto(reportRaw); err != nil {
			panic(fmt.Errorf("parsing report: %w", err))
		}

		log.Println("Success!")
		time.Sleep(5 * time.Second)
	}
}
