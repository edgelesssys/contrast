package manifest

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
				TrustedIDKeyHashes: []HexString{
					"b2bcf1b11d9fb3f2e4e7979546844d26c30255fff0775f3af56f8295f361a7d1a34a54516d41abfff7320763a5b701d8",
					"22087e0b99b911c9cffccfd9550a054531c105d46ed6d31f948eae56bd2defa4887e2fc4207768ec610aa232ac7490c4",
				},
			},
		},
	}
}
