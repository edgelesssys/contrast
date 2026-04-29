// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package insecure provides a fake aTLS issuer and validator for development
// platforms without confidential computing hardware.
package insecure

import (
	"context"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/edgelesssys/contrast/internal/oid"
)

// HostdataAddr is the address where the initdata-processor serves the
// hostdata digest on insecure platforms.
const HostdataAddr = "127.0.0.1:19629"

// HostdataURL is the full URL for fetching the hostdata digest.
const HostdataURL = "http://" + HostdataAddr + "/hostdata"

// Issuer issues fake attestation documents for insecure (non-CC) platforms.
//
// It fetches the initdata digest from the local initdata-processor HTTP server
// and packages it with the report data into a JSON attestation document.
type Issuer struct {
	hostdataURL string
	client      *http.Client
}

// NewIssuer creates a new insecure issuer.
func NewIssuer() *Issuer {
	return &Issuer{hostdataURL: HostdataURL, client: http.DefaultClient}
}

// OID returns the OID for the insecure attestation.
func (i *Issuer) OID() asn1.ObjectIdentifier {
	return oid.RawInsecureReport
}

// Issue creates a fake attestation document containing the report data and
// the initdata digest fetched from the local hostdata server.
func (i *Issuer) Issue(ctx context.Context, reportData [64]byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.hostdataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating hostdata request: %w", err)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching hostdata from %q: %w", i.hostdataURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching hostdata: status %s", resp.Status)
	}
	hostData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading hostdata response: %w", err)
	}
	return json.Marshal(attestationDoc{
		ReportData: reportData[:],
		HostData:   hostData,
	})
}

// attestationDoc is the fake attestation document exchanged between issuer and validator.
type attestationDoc struct {
	ReportData []byte `json:"reportData"`
	HostData   []byte `json:"hostData"`
}
