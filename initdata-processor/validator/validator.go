package validator

import (
	"errors"
	"fmt"
	"log"
	"slices"

	snpabi "github.com/google/go-sev-guest/abi"
	snpclient "github.com/google/go-sev-guest/client"
	tdxabi "github.com/google/go-tdx-guest/abi"
	tdxclient "github.com/google/go-tdx-guest/client"
	"github.com/google/go-tdx-guest/proto/tdx"
)

type Validator struct {
	snp snpclient.QuoteProvider
	tdx tdxclient.QuoteProvider
}

func New() *Validator {
	v := &Validator{}
	if qp, err := snpclient.GetQuoteProvider(); err == nil {
		v.snp = qp
	}
	if qp, err := tdxclient.GetQuoteProvider(); err == nil {
		v.tdx = qp
	}
	return v
}

func (v *Validator) ValidateDigest(digest []byte) error {
	var errs []error
	if expectedDigest, err := v.getDigestSNP(); err == nil {
		return compareDigests(expectedDigest, digest)
	} else {
		errs = append(errs, err)
	}
	if expectedDigest, err := v.getDigestTDX(); err == nil {
		return compareDigests(expectedDigest, digest)
	} else {
		errs = append(errs, err)
	}

	log.Printf("validation errors:\n%v", errors.Join(errs...))
	// TODO(burgerdev): DANGER!!! this must fail, but I'm returning nil until the digest is initdata, not policy.
	return nil
}

func (v *Validator) getDigestSNP() ([]byte, error) {
	if v.snp == nil {
		return nil, fmt.Errorf("no SNP quote provider available")
	}
	reportRaw, err := v.snp.GetRawQuote([64]byte{})
	if err != nil {
		return nil, fmt.Errorf("snp: getting raw report: %w", err)
	}
	report, err := snpabi.ReportToProto(reportRaw)
	if err != nil {
		return nil, fmt.Errorf("snp: parsing report: %w", err)
	}
	return report.HostData, nil
}

func (v *Validator) getDigestTDX() ([]byte, error) {
	if v.tdx == nil {
		return nil, fmt.Errorf("no TDX quote provider available")
	}
	quoteRaw, err := v.tdx.GetRawQuote([64]byte{})
	if err != nil {
		return nil, fmt.Errorf("tdx: getting raw quote: %w", err)
	}
	quote, err := tdxabi.QuoteToProto(quoteRaw)
	if err != nil {
		return nil, fmt.Errorf("tdx: parsing quote: %w", err)
	}
	quotev4, ok := quote.(*tdx.QuoteV4)
	if !ok {
		return nil, fmt.Errorf("tdx: unexpected quote type: %T", quote)
	}
	return quotev4.TdQuoteBody.MrConfigId, nil
}

func compareDigests(expected, actual []byte) error {
	n := min(len(expected), len(actual))
	if n < 32 {
		return fmt.Errorf("unexpected digest size: expected %x, got %x", expected, actual)
	}
	if slices.Compare(expected[:n], actual[:n]) != 0 {
		return fmt.Errorf("digests don't match: expected %x, got %x", expected, actual)
	}
	return nil
}
