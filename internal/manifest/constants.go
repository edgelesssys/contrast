package manifest

// This value is injected at build time.
var trustedMeasurement = "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

// Default returns a default manifest.
func Default() Manifest {
	return Manifest{
		ReferenceValues: ReferenceValues{
			SNP: SNPReferenceValues{
				MinimumTCB: SNPTCB{
					BootloaderVersion: 3,
					TEEVersion:        0,
					SNPVersion:        8,
					MicrocodeVersion:  115,
				},
				TrustedMeasurement: HexString(trustedMeasurement),
			},
		},
	}
}
