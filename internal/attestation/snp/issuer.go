/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package snp

import (
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/google/go-sev-guest/client"
)

type Issuer struct {
	snpDevicePath string
}

// NewIssuer returns a new Issuer.
func NewIssuer() *Issuer {
	return &Issuer{}
}

func (i *Issuer) OID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 3, 9900, 77, 77}
}

// userData is hash of issuer public key.
// nonce from validator.
func (i *Issuer) Issue(ctx context.Context, ownPublicKey []byte, nonce []byte) (res []byte, err error) {
	log.Println("issuer: issue called")
	defer func() {
		if err != nil {
			log.Printf("Failed to issue attestation statement: %s", err)
		}
	}()

	snpGuestDevice, err := client.OpenDevice()
	if err != nil {
		log.Fatalf("issuer: opening device: %v", err)
	}
	defer snpGuestDevice.Close()

	reportData := constructReportData(ownPublicKey, nonce)

	reportRaw, err := client.GetRawReport(snpGuestDevice, reportData)
	if err != nil {
		return nil, fmt.Errorf("getting raw report: %w", err)
	}
	log.Printf("issuer: Report raw: %v", hex.EncodeToString(reportRaw))

	reportB64 := make([]byte, base64.StdEncoding.EncodedLen(len(reportRaw)))
	base64.StdEncoding.Encode(reportB64, reportRaw)

	log.Println("issuer: Successfully issued attestation statement")
	return reportB64, nil
}
