// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validator

import (
	"fmt"

	"github.com/google/go-sev-guest/abi"
	"github.com/google/go-sev-guest/client"
)

// snpDigestGetter obtains the expected initdata digest from HOSTDATA.
//
// The SEV-SNP ABI spec [1] (rev. 1.58, chapter 7) states that guest messages are protected against
// the hypervisor reading, altering, dropping or replaying them, through encryption with VMPCKs and
// sequence numbering. Reports are created using guest messages, thus we don't need to verify the
// report locally.
//
// [1]: https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/specifications/56860.pdf
type snpDigestGetter struct {
	client.QuoteProvider
}

func (g *snpDigestGetter) GetDigest() ([]byte, error) {
	reportRaw, err := g.GetRawQuote([64]byte{})
	if err != nil {
		return nil, fmt.Errorf("snp: getting raw report: %w", err)
	}
	report, err := abi.ReportToProto(reportRaw)
	if err != nil {
		return nil, fmt.Errorf("snp: parsing report: %w", err)
	}
	return report.HostData, nil
}
