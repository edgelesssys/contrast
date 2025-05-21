// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package issuer

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/google/go-sev-guest/proto/sevsnp"
)

// source: https://learn.microsoft.com/en-us/azure/security/fundamentals/trusted-hardware-identity-management#uri-parameters
const thimCertificationURL = "http://169.254.169.254/metadata/THIM/amd/certification"

// THIMSNPCertification represents a cert chain for SNP.
// The chain contains:
// - VCEK certificate
// - ASK certificate
// - ARK (root) certificate
//
// Source:
// https://learn.microsoft.com/en-us/azure/security/fundamentals/trusted-hardware-identity-management#definitions .
type THIMSNPCertification struct {
	VCEKCert         string `json:"vcekCert"`
	TCBM             string `json:"tcbm"`
	CertificateChain string `json:"certificateChain"`
	CacheControl     string `json:"cacheControl,omitempty"`
}

// Proto returns the certificate chain as a go-sev-guest proto.
func (c THIMSNPCertification) Proto() (*sevsnp.CertificateChain, error) {
	vcekCert, rest := pem.Decode([]byte(c.VCEKCert))
	if vcekCert == nil || len(rest) != 0 {
		return nil, fmt.Errorf("decoding certification: missing or unexpected trailing data in VCEK certificate")
	}
	askCert, rest := pem.Decode([]byte(c.CertificateChain))
	if askCert == nil {
		return nil, fmt.Errorf("decoding certification: missing ASK certificate")
	}
	arkCert, rest := pem.Decode(rest)
	if arkCert == nil {
		return nil, fmt.Errorf("decoding certification: missing ARK certificate")
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("decoding certification: unexpected trailing data")
	}

	chain := &sevsnp.CertificateChain{
		VcekCert: vcekCert.Bytes,
		AskCert:  askCert.Bytes,
		ArkCert:  arkCert.Bytes,
	}

	return chain, nil
}

// THIMGetter is a getter for the THIM certification.
type THIMGetter struct {
	httpClient httpClient

	cachedResponse []byte
	validUntil     time.Time

	mux sync.RWMutex
}

// NewTHIMGetter returns a new THIMGetter.
func NewTHIMGetter(httpClient httpClient) *THIMGetter {
	return &THIMGetter{httpClient: httpClient}
}

// GetCertification returns the THIM certification.
func (t *THIMGetter) GetCertification(ctx context.Context) (THIMSNPCertification, error) {
	// Return cached response if it is still valid.
	if cached := t.getCached(); cached != nil {
		var certification THIMSNPCertification
		if err := json.Unmarshal(cached, &certification); err != nil {
			return THIMSNPCertification{}, fmt.Errorf("unmarshalling cached THIM certification: %w", err)
		}
		return certification, nil
	}

	// Fetch fresh certification.
	uri, err := url.Parse(thimCertificationURL)
	if err != nil {
		return THIMSNPCertification{}, fmt.Errorf("parsing THIM certification URL: %w", err)
	}
	// source:
	// https://learn.microsoft.com/en-us/azure/security/fundamentals/trusted-hardware-identity-management#how-do-i-request-collateral-in-a-confidential-virtual-machine
	req := &http.Request{
		Method: http.MethodGet,
		URL:    uri,
		Header: http.Header{
			"Metadata": {"true"},
		},
	}
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := t.httpClient.Do(req.WithContext(reqCtx))
	if err != nil {
		return THIMSNPCertification{}, fmt.Errorf("getting THIM certification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return THIMSNPCertification{}, fmt.Errorf("getting THIM certification: unexpected status code %d", resp.StatusCode)
	}
	rawCertification, err := io.ReadAll(resp.Body)
	if err != nil {
		return THIMSNPCertification{}, fmt.Errorf("reading THIM certification: %w", err)
	}

	var certification THIMSNPCertification
	if err := json.Unmarshal(rawCertification, &certification); err != nil {
		return THIMSNPCertification{}, fmt.Errorf("unmarshalling THIM certification: %w", err)
	}

	// Cache the response.
	var cacheControl int64
	if certification.CacheControl != "" {
		var err error
		cacheControl, err = strconv.ParseInt(certification.CacheControl, 10, 64)
		if err != nil {
			return THIMSNPCertification{}, fmt.Errorf("parsing cache control duration: %w", err)
		}

	} else {
		cacheControl = 86400 // Default to 1 day (this is the observed behavior of the THIM).
	}
	t.setCached(rawCertification, time.Now().Add(time.Duration(cacheControl)*time.Second))

	return certification, nil
}

// getCached returns the cached THIM certification.
// The method returns nil if the certification is not cached or expired.
func (t *THIMGetter) getCached() []byte {
	t.mux.RLock()
	defer t.mux.RUnlock()

	if t.cachedResponse == nil || time.Now().After(t.validUntil) {
		return nil
	}
	return t.cachedResponse[:]
}

// setCached sets the cached THIM certification.
func (t *THIMGetter) setCached(certification []byte, validUntil time.Time) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.cachedResponse = certification
	t.validUntil = validUntil
}

// httpClient represents the ability to fetch data from the internet from an HTTP URL.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
