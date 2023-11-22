/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package snp

import (
	"bytes"
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/google/go-sev-guest/abi"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) OID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 3, 9900, 77, 77}
}

// Validate a TPM based attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	log.Println("validator: validate called")
	defer func() {
		if err != nil {
			log.Printf("Failed to validate attestation document: %s", err)
		}
	}()

	log.Printf("validator: Nonce: %v", hex.EncodeToString(nonce))

	reportRaw := make([]byte, base64.StdEncoding.DecodedLen(len(attDocRaw)))
	if _, err = base64.StdEncoding.Decode(reportRaw, attDocRaw); err != nil {
		return err
	}
	log.Printf("validator: Report raw: %v", hex.EncodeToString(reportRaw))

	report, err := abi.ReportToProto(reportRaw)
	if err != nil {
		log.Fatalf("converting report to proto: %v", err)
	}

	if err := abi.ValidateReportFormat(reportRaw); err != nil {
		return fmt.Errorf("validating report format: %w", err)
	}

	reportDataExpected := constructReportData(peerPublicKey, nonce)
	if !bytes.Equal(report.ReportData, reportDataExpected[:]) {
		return errors.New("certificate hash does not match user data")
	}

	log.Println("validator: Successfully validated attestation document")
	return nil
}
