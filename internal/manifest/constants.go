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
					"bb4bb49681f267bd1d504ce1c4388abcf7e3e53b6003a1bfcfe9884056047912ebb9a813da95cf711a0410ddc00fe65b", // Added 2024-01-22
					"92898fbc330c89f8a38b8516087970b1d3361e017c84bd5abe901cab7edeb0a4271509edba1670c14feb82293bcde33f", // Added 2024-02-07
					"089ee8adfc810a72eb2683007f34db9f8160c4d1936b70570b779ef3b7bb66046194298cea8d51ebfd4b7c8a2b8ea2d7", // Added 2024-02-21
					"1383573d02170f77b1fc2a8bfd5eaec89b0158b3f186eee7b65f785187c41b50be5d97e3b23fa9c5a4b70fe0d1e03af7", // Added 2024-03-12
				},
			},
		},
	}
}
