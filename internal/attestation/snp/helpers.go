package snp

import "crypto/sha512"

func constructReportData(publicKey []byte, nonce []byte) [64]byte {
	return sha512.Sum512(append(publicKey, nonce...))
}
