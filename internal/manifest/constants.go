// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

// TrustedMeasurement contains the expected launch digest and is injected at build time.
var TrustedMeasurement = "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

// Default returns a default manifest.
func Default() Manifest {
	return Manifest{
		ReferenceValues: ReferenceValues{
			TrustedMeasurement: HexString(TrustedMeasurement),
		},
	}
}

// DefaultAKS returns a default manifest with AKS reference values.
func DefaultAKS() Manifest {
	mnfst := Default()
	mnfst.ReferenceValues.SNP = SNPReferenceValues{
		MinimumTCB: SNPTCB{
			BootloaderVersion: toPtr(SVN(3)),
			TEEVersion:        toPtr(SVN(0)),
			SNPVersion:        toPtr(SVN(8)),
			MicrocodeVersion:  toPtr(SVN(115)),
		},
	}
	return mnfst
}

func toPtr[T any](t T) *T {
	return &t
}
