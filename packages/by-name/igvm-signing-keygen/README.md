# IGVM signing `keygen`

Build instructions for generating a IGVM signing key. This package contains a
python script for generating valid signing keys for the ID Block of AMD
SEV-SNP-enabled VMs. Such a signing key is needed for the `kata-igvm` package.

# IGVM `snakeoil` key

The `snakeoil` key (`igvn-signing-keygen.snakeoilPem`) is a well-known key used
to generate reproducible IGVM files. In Contrast, we verify the launch digest of
SNP-enabled pod-VMs. This means it's perfectly fine for this key to be known to
the public. In exchange, this allows any third party to produce the same IGVM
files from source (bit-by-bit reproducibility).

The term `snakeoil` is commonly used in this context to refer to a key that
exists for tools that expect it, without adhering to the same security standards
(keeping it secret).
