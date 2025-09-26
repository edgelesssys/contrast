// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validator

import (
	"fmt"

	"github.com/google/go-tdx-guest/abi"
	"github.com/google/go-tdx-guest/client"
	"github.com/google/go-tdx-guest/proto/tdx"
)

// tdxDigestGetter obtains the expected initdata digest from MRCONFIGID.
//
// On TDX, the guest TD obtains a report directly from the TDX module using a TDCALL. This TDCALL
// can't be observed by the hypervisor because request and response are exchanged on encrypted TD
// memory. Documentation for this is somewhat scattered, but a good starting point may be the TDX
// module spec [1], section 24.3.3.
//
// [1]: https://cdrdv2-public.intel.com/733568/tdx-module-1.0-public-spec-344425005.pdf
type tdxDigestGetter struct {
	client.QuoteProvider
}

func (g *tdxDigestGetter) GetDigest() ([]byte, error) {
	quoteRaw, err := g.GetRawQuote([64]byte{})
	if err != nil {
		return nil, fmt.Errorf("tdx: getting raw quote: %w", err)
	}
	quote, err := abi.QuoteToProto(quoteRaw)
	if err != nil {
		return nil, fmt.Errorf("tdx: parsing quote: %w", err)
	}
	quotev4, ok := quote.(*tdx.QuoteV4)
	if !ok {
		return nil, fmt.Errorf("tdx: unexpected quote type: %T", quote)
	}
	return quotev4.TdQuoteBody.MrConfigId, nil
}
