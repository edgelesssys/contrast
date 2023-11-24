/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package snp

import (
	"context"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
)

type Validator struct {
	validateOptsGen validateOptsGenerator
}

type validateOptsGenerator interface {
	SNPValidateOpts(report *sevsnp.Report) (*validate.Options, error)
}

func NewValidator(optsGen validateOptsGenerator) *Validator {
	return &Validator{
		validateOptsGen: optsGen,
	}
}

func (v *Validator) OID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 3, 9900, 77, 77}
}

// Validate a TPM based attestation.
func (v *Validator) Validate(ctx context.Context, attDocRaw []byte, nonce []byte, peerPublicKey []byte) (err error) {
	log.Printf("validator: validate called with nonce %s", hex.EncodeToString(nonce))
	defer func() {
		if err != nil {
			log.Printf("Failed to validate attestation document: %s", err)
		}
	}()

	// Parse the attestation document.

	reportRaw := make([]byte, base64.StdEncoding.DecodedLen(len(attDocRaw)))
	if _, err = base64.StdEncoding.Decode(reportRaw, attDocRaw); err != nil {
		return err
	}
	log.Printf("validator: Report raw: %v", hex.EncodeToString(reportRaw))

	report, err := abi.ReportToProto(reportRaw)
	if err != nil {
		log.Fatalf("converting report to proto: %v", err)
	}

	// Report signature verification.

	verifyOpts := &verify.Options{}
	attestation, err := verify.GetAttestationFromReport(report, verifyOpts)
	if err := verify.SnpAttestation(attestation, verifyOpts); err != nil {
		log.Fatalf("verifying report: %v", err)
	}
	log.Println("validator: Successfully verified report signature")

	// Validate the report data.

	reportDataExpected := constructReportData(peerPublicKey, nonce)
	validateOpts, err := v.validateOptsGen.SNPValidateOpts(report)
	if err != nil {
		return fmt.Errorf("generating validation options: %w", err)
	}
	validateOpts.ReportData = reportDataExpected[:]
	if err := validate.SnpAttestation(attestation, validateOpts); err != nil {
		return err
	}
	log.Println("validator: Successfully validated report data")

	log.Println("validator: done")
	return nil
}
