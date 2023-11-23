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
	"log"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/google/go-sev-guest/validate"
	"github.com/google/go-sev-guest/verify"
)

type Validator struct {
	callbacks []ValidateCallbacker
}

type ValidateCallbacker interface {
	ValidateCallback(ctx context.Context, report *sevsnp.Report, nonce []byte, peerPublicKey []byte) error
}

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
	validateOpts := &validate.Options{
		GuestPolicy: abi.SnpPolicy{
			Debug: false,
			SMT:   true,
		},
		VMPL:                      new(int),
		PermitProvisionalFirmware: true,
		ReportData:                reportDataExpected[:],
	}
	if err := validate.SnpAttestation(attestation, validateOpts); err != nil {
		return err
	}
	log.Println("validator: Successfully validated report data")

	log.Println("validator: done")
	return nil
}
