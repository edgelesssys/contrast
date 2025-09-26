// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package validator

import (
	"errors"
	"fmt"
	"slices"

	snpclient "github.com/google/go-sev-guest/client"
	tdxclient "github.com/google/go-tdx-guest/client"
)

// Validator compares an incoming initdata hash with HOSTDATA or MRCONFIGID, depending in the platform.
//
// Instances need to be created with New.
type Validator struct {
	digestGetter digestGetter
}

// New constructs a Validator suitable for the current runtime environment, if available.
func New() (*Validator, error) {
	sqp, serr := getSNPQuoteProvider()
	if serr == nil {
		return &Validator{&snpDigestGetter{sqp}}, nil
	}
	tqp, terr := getTDXQuoteProvider()
	if terr == nil {
		return &Validator{&tdxDigestGetter{tqp}}, nil
	}
	return nil, fmt.Errorf("%w:\nTDX:%w\nSNP:%w", errBadPlatform, terr, serr)
}

// ValidateDigest compares the given digest with either MRCONFIGID or HOSTDATA, and returns an error if they don't match.
//
// The minimum size of the digest argument is 32 bytes, corresponding to a SHA256 hash.
// The comparison matches the implementation in the Kata runtime with respect to truncation and
// padding:
//   - https://github.com/kata-containers/kata-containers/blob/0d58bad/src/runtime/pkg/govmm/qemu/qemu.go#L427
//   - https://github.com/kata-containers/kata-containers/blob/0d58bad/src/runtime/pkg/govmm/qemu/qemu.go#L516
func (v *Validator) ValidateDigest(digest []byte) error {
	expectedDigest, err := v.digestGetter.GetDigest()
	if err != nil {
		return err
	}
	return compareDigests(expectedDigest, digest)
}

type digestGetter interface {
	GetDigest() ([]byte, error)
}

func compareDigests(expected, actual []byte) error {
	// minDigestLength is the minimum length of a digest.
	// We check the length here just for completeness. The intended use of this function is to
	// compare initdata digests (which are sha256, sha384 or sha512) with either HOSTDATA (32
	// bytes) or MRCONFIGID (48 bytes), so all byte slices should have a minimum size of 32
	// anyway.
	const minDigestLength = 32
	n := min(len(expected), len(actual))
	if n < minDigestLength {
		return fmt.Errorf("%w: expected %x, got %x", errUnexpectedDigestSize, expected, actual)
	}
	if slices.Compare(expected[:n], actual[:n]) != 0 {
		return fmt.Errorf("%w: expected %x, got %x", errDigestMismatch, expected, actual)
	}
	return nil
}

func getSNPQuoteProvider() (snpclient.QuoteProvider, error) {
	// snpclient checks IsSupported internally.
	return snpclient.GetQuoteProvider()
}

func getTDXQuoteProvider() (tdxclient.QuoteProvider, error) {
	tqp, err := tdxclient.GetQuoteProvider()
	if err != nil {
		return nil, err
	}
	if err := tqp.IsSupported(); err != nil {
		return nil, err
	}
	return tqp, nil
}

var (
	errBadPlatform          = errors.New("no digest getter available for current platform")
	errUnexpectedDigestSize = errors.New("unexpected digest size")
	errDigestMismatch       = errors.New("digests don't match")
)
