// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSingleKey(t *testing.T) {
	bits := []int{2048, 4096}
	seeds := [][]byte{{}, {1, 2, 3, 4, 5, 6, 7, 8}}

	for _, b := range bits {
		for numKeys := range 3 {
			for _, seed := range seeds {
				name := fmt.Sprintf("bits=%d numKeys=%d seed=[%d]byte", b, numKeys, len(seed))
				t.Run(name, func(t *testing.T) {
					require := require.New(t)
					keys := make([]*rsa.PrivateKey, numKeys)
					pubKeys := make([]HexString, numKeys)
					for i := range numKeys {
						keys[i] = getTestKey(t, b, i)
						pubKeys[i] = MarshalSeedShareOwnerKey(&keys[i].PublicKey)
					}

					seedShares, err := EncryptSeedShares(seed, pubKeys)
					require.NoError(err)
					require.Len(seedShares, numKeys)

					for i := range numKeys {
						decryptedSeedShare, err := DecryptSeedShare(keys[i], seedShares[i])
						require.NoError(err)

						require.Equal(seed, decryptedSeedShare)
					}
				})
			}
		}
	}

	t.Run("decrypting with an unrelated key should fail", func(t *testing.T) {
		require := require.New(t)

		rightKey := getTestKey(t, 2048, 1)
		wrongKey := getTestKey(t, 2048, 2)

		seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}

		pubKeyHex := MarshalSeedShareOwnerKey(&rightKey.PublicKey)

		seedShares, err := EncryptSeedShares(seed, []HexString{pubKeyHex})
		require.NoError(err)
		require.Len(seedShares, 1)

		decryptedSeedShare, err := DecryptSeedShare(wrongKey, seedShares[0])
		require.Error(err)
		require.Nil(decryptedSeedShare)
	})
}

func TestDecryptSeedShare_WrongLabel(t *testing.T) {
	require := require.New(t)

	key := getTestKey(t, 2048, 1)

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	pubKeyHex := MarshalSeedShareOwnerKey(&key.PublicKey)

	cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, seed, []byte("this-label-is-wrong-for-contrast-seeds"))
	require.NoError(err)
	seedShare := &userapi.SeedShare{
		EncryptedSeed: cipherText,
		PublicKey:     pubKeyHex.String(),
	}

	_, err = DecryptSeedShare(key, seedShare)
	require.Error(err)
}

func TestSeedShareKeyParseMarshal(t *testing.T) {
	keyData, err := NewSeedShareOwnerPrivateKey()
	require.NoError(t, err)

	key, err := ParseSeedshareOwnerPrivateKey(keyData)
	require.NoError(t, err)

	pubHexStr := MarshalSeedShareOwnerKey(&key.PublicKey)

	_, err = ParseSeedShareOwnerKey(pubHexStr)
	require.NoError(t, err)

	publicKeyHexStr, err := ExtractSeedshareOwnerPublicKey(keyData)
	require.NoError(t, err)

	publicKeyBytes, err := publicKeyHexStr.Bytes()
	require.NoError(t, err)
	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	publicKeyHexStrReparsed, err := ExtractSeedshareOwnerPublicKey(publicKeyPem)
	require.NoError(t, err)
	assert.Equal(t, publicKeyHexStr, publicKeyHexStrReparsed)
}

func TestWorkloadOwnerKeyParseMarshal(t *testing.T) {
	keyData, err := NewWorkloadOwnerKey()
	require.NoError(t, err)

	privateKey, err := ParseWorkloadOwnerPrivateKey(keyData)
	require.NoError(t, err)

	keyDigest := HashWorkloadOwnerKey(&privateKey.PublicKey)
	assert.Len(t, keyDigest, 64)

	publicKeyBytes, err := ExtractWorkloadOwnerPublicKey(keyData)
	require.NoError(t, err)

	publicKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	publicKeyBytesReparsed, err := ExtractWorkloadOwnerPublicKey(publicKeyPem)
	require.NoError(t, err)
	assert.Equal(t, publicKeyBytes, publicKeyBytesReparsed)
}

func getTestKey(t *testing.T, keyLen int, id int) *rsa.PrivateKey {
	t.Helper()

	keys2048 := []string{
		`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAo/5d1OLkIcw55Lk66qBP2Bf4qu5KSnBDu1Rt87MvHlL/KFHK
Wy3qtHXxT1/Ofme7iUvZJfwHAo7SZJ0XDt1EvAXMO930Rg/jbeygZ2QcHDkJt86g
lIqfEyqY8Qhescy/sR+R6p/mdDzqkgYU4pVp72BG1P0NlW2MeMjesswEIgclUS5+
RyysrLhvG0v8pwRAqbsOQSHIt3bCq66TBlmY5BwnyM0IfsOpUhWWB3t09DyzGXXf
ZlG/NTfdCp53A6O7c22YWx1HbFwI0p1VZVKqY2WkW4V7AI9IHkoP2X0q+GTGcuht
EGPDuA2Lm5xNWsjcZGAh/3SiRnnSXLn4a5ygwwIDAQABAoIBAC5nG0XRrZuinf8K
KpGZKv6KSeKT6pGFkqS/Dx2V5g8+jNRr0EZch7zoYo+DHHrH/1iqDZeh6Jngr8eo
a43ZLknFmjSWaTgp5sCD5B9dRFb4DLflz6o4TyqtMvmA1MqalOMZe8BX3m2ljvoi
nmY+wOrq4yABOoa9qLHEpL8S21iFGSR7RLXQI5fYnCLvQMC5KYou9C8FS1kmGXqH
8cTMqvnC3clbeA77oXnEH1QTEi9SRW7ZpweD6gIk3g9N1NX/WSefvltRErUkkUYX
2z4iKpf/RvZo6R1udSMteTT780Zuh0FBa52m0OaXUa1hwwXJGgzix4bH8L3cuk8s
QKHFaoECgYEAzs1IYUMGipuUebb4Byv11CaWskX37AFwAyfnRTkoGKoZrG7lSWUX
bUEs2stDEUh8OPkIPiTN3yRgffitYzJU6vq7PuWGplJqkkdeIKEbekP1gWC/IOAa
ysTg7xaWUJ1AyJGzQlkBfZ9E+Ov2J1pfXLJAlbMlJtfNAfIiQrnFKOcCgYEAywH1
48sYTwQpNGd74yXhcYvcJ3CKduglagaFF0oDHoJEa7iI+Cn3jGRTEe2NyJR/eQG9
EMjDd7oNCttw6On9GbUYUEAkrrXIu4g/2a5iXCiUJ4ogstVAU5SSCmskJMJ80UPS
QXrURJYhQKy659SCrnCQzHE0blH+3mNvEgzCwcUCgYAKK5BOsDQnJuWTYssp1yCc
0VUB6Wz630s57IF0Jw5wwBTJJR8DkAQp7FWfYPWoaO8rAhxEqhyxx6EzMMKeKUCB
2djRjAomLdFt5jKb2jB2v9bYCQD9RegrZqlFONAloMYp1viA382x9t42e6w8XTZp
YZ7Jfejq0xwS52yF6YrnqwKBgB/6yxV7ZPTtnuAWfTmnOgB9G81KuUVKBLrTFBw7
GIqx0r11cH5Hfiurkjp8xZ0XZ41UbMg8GC7ALFXNg9ftJGXsVUwvDphHIrwIFqbg
Bbam3c/svoHtfhisiuUBQ8xWpvsASBrwkofLbqmVjEwA+iUormbGXpAScqft2g1p
3TRRAoGAYgIoU08FjEzuduYAg9mPQ/dskmZrP1rOqkbWm6KOJurkS3ycvl9A7JLg
XKCxVHVi2YAvJZHQlOAoJtVkVJ81qq5WJtPmxE+DQY5KXUCAWRlUUR3jEbFCM3Id
137Svhr7MrikCXGVQv4+liDa8OGP5uU17qXx2C33VWnjckwTVco=
-----END RSA PRIVATE KEY-----`,
		`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzdBQPBa04M8a8mf9hYYl8uCYGReWsGfkIYebTJJKG0jySEi4
1//nUQGpt267++ayYAJktDGiikOlpPWJu9zFhUMoT16zPdlzsumgjuYr9Y9TYevH
cFqlXUJhXdV+wL7l7v3bR692R3pFDhNk2GrFi7et4EMYUKJi2N8vJdeawLDzV0aP
AR8tutaOYRkSI0AnMEgz78Z7L0dhdgq4Kfrw20kCy9xz+L0f9IEA9fr2xcSQwByM
qXeof5g+kmuLdvK68jCdOGaUzZBnG2pErDBg1bNRQOMBiCiYMo5xesgk7Gb5bGLE
D+bTOaihTlRI0OBZmmj76Sb1WvueWD3Kbnm1hwIDAQABAoIBAFp0Gj8+b4J6I0q0
P2zml1kWMmKcxmKDVnUSB0Pw61bwiWMRawOreXtVssRmi4HbUzv08VNsmRYRQwSr
0TvafIjkChxP75DYOAxCt4j2Sg2jTy8zE7UicZj6Kpa11P5bJ+0QbsYjrGUfrKfS
CDlBO877DBULB+2wYKcV97+28VGL6XCVfFG2za6707qlu0XD0JvmZvM7ll+fM/+a
o9weMNa5/QJ82UDhu9Jw7tL8AlfIF+3ouSnbF0MUqb16N9KXZYAXjWPl/W3ltLrD
mq2oNAdKs7zvX+nAo1RFrdoQ5GwNUkA8gpMwt6aJQaITav7s6NsZJNEeowvH7wgu
ZLZlfCkCgYEA8XvtqFV6SIrSQp9bwwruS6HqYzk8qjuq6gNNpiMDJT6HXqoEcXqm
wJy+nIHFNWm2+YAMgDCtRmsV1Jkc/BVzDbylqAkJYfLhHfiGNg+Eu6PLMJ9diTxR
IQnhocJ7ckqiyPZMFpMMRxO1hMwk0Dpv0x+kZ5LuLBE6JJA2v8aCkP0CgYEA2i94
9fcDUUTxgvSp1g3m/EZ2doMpQILu8P4oih+fafF4GHiuMbUxYG85ynrHmFEbrkmu
u9gO8Z6DdJf+MUSVkRwmQLqL0m4CqHkp0JSgwF60WI6mtlVGmY5ctkReBmR+uihU
daf8AM80NtTsS5EregcHYdL8rO6J7XdP5/WZmdMCgYEAs0lQVGdKB6vRmZcZGMD3
1P1cuNhY+wabyWw0bUGXZ0J6XMUb0Wi/f0egmTAby6E2MR1pqo75RsvghFw9UcdX
CX7i+tPivG8Hximq814oLOvZwrq/RlGa5k2g0GlFH8DcBRofua7pMagnX3X1awfH
2FaoyCElZWBQ666Kh22JqL0CgYEAjnWmrjrtkJfKdW4YomoLYrcDTFhRjAyxxOq+
P4lsRfljJ99MJaqgJc4Z1soaHqr+vurfS0lEYKDWRj+jujmEyu2tUGA9QVWRKL2L
/uO8nj531Ma3tZ+ybDrW8C9tkRD26cfBdd0MHt8rwY1/B8wurgt+13Gyh5tstX9M
zjC/bP8CgYEAl1J7yiJPZx2gK6DPugOSYmoykuwzxWHmYYW3HgEfMLN1eACgdigO
CWq7oQW3s+TXj2jZ5TFaOzmwMdKAHKC9vxO2GQzhc6dGyMN16PBB4MBimcWEIyyQ
7omXBzyRWa+Av/OOkHYtKmO3nYwVJbGE3cNTi7fcnsWo6YqZnq+6pec=
-----END RSA PRIVATE KEY-----`,
		`-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEAzE0YuRJuzphafwZIDdNSCcksdNyuTLTiAJDOHIvhYCLDShq/
pCAUPhKc1tJ08LJmX8bGHpjdhBIsf7kc3HlAJDz+FvgDEmStt3vVC9wf03TRwegH
QHOjVK+Ld6fYaeaEzBsHPeGUdrWvMkiJvUowfl/LNQEhlpXnu5XQ+DDo4gbvlzoA
Ne0BD9nbqO/6F+WgyscOFo78oSjjH8OzoO2Vwk0e11kPnk7acOdFHagUcJzJ+z+/
Gt8FzC+cIY7q/Gm24ClGEramyVh1HLV5a/loy9Iei+Zk+j9gf677kvCgX15RDlsb
BMgtmMASAU4iYg7Tzn+iXj2LMHa8O5eKIcVl8QIDAQABAoIBAQCA1JEli7EiSEKw
3VYsmbifASQuoUasilgGAUpoB/FcPS+nGE0HA0+ggP40HUyux+D0vpUKkF0Hvqoe
9K11fmRrNacI9uaA/8nn7YfATdQn9P+c5mNESmeRrI0nLEm1Ji9Rwi2D4S9M2G8d
J07sdc80sdWjnA4BNpCF4wh+qeEBPjv0G7kBYwdWwj9mzBCPReQYyiiQdi55XlpR
cLvs2I80+dU4/FJB+c9LMn8FxzDtPjk3X/e71EK9O7v4kRcai6rlhpIfeWRpveSK
G+c2T6lNJflar77OeV0IUjWHcjFAD/4n1BqSII1OtxahqCPCZqZNmpLemXJUuvXq
f/KGZlndAoGBAOl3uTwMqNR83fqLFrXv1yWuMPvTuiR+xSd3pt+3U8CLLdXnJl+q
IP5tdIgiefEon9QfMi7ZahM98roGs4vEUbc1FV1xUHKtYhUY8Pb1IpV3wQnx1DfU
eS3w5z39X/YothncMPciru2BaLxlCVTAjEgVqWfJjy/o7D7xCiKKNecfAoGBAOAE
wdUiKTelIVFiqRVOLk/oMqD9pyEYjcVVclvwqN3Y3dH68sIzcwaITx/rAUw0n4nP
uyH1LYV61wUHEkHabzIm+JStGPJMTWfv/fYrfwuxe/aijPeCDww/KS5R7dnBM6PX
pI8+qx8z83oakd3NJ2IZgLLPCAKMYoMnkRJXWWDvAoGBAM6eNkD+syvalll1XtQF
PtMKJi+4YbSKvNEBA55aELUGd7omp79iQXDqTYdte54B5fFE6pSrtUTyPi6EX8IC
LI+HWzEnZ5sV9wfU2uy0ZbcCFMVIUBhY4iXWXdBuvM6NmRup02vkNgvby2VvxaJM
BdqF0TcZGq+749iQWffXeXzhAoGBALv74fgLQZE7Vbko1IBXac+OJyYnlJ7WLumg
KWXzjpETkhjJv/qtF+IsclFzcFRVeGc51WvKhVeUXGkQpQZz2Ym5YDHLC7sPwojs
wC1aFLNoTYEKqMZt8lixi8od4D0xvjbIF4RI72owuykEsNDyfhD5G6Fwz+TrjyNG
CZvdhtgrAoGBAIMH9oKhGKmfA3xHZ3P3uURHFvD/YbaTjAe9pA+7EqY+dEfM0s7m
vBPtBcukDv9utMHDOdsB6P/squG/4xUlhXeK/IjGAOLueYkbkUkJ9cGt8VnmlKfx
8K6KhRbIBGL7uY0GMhKV4OuCgBpMDpnYfi+lzJwZq4u2dJqFcGk5IT52
-----END RSA PRIVATE KEY-----`,
	}
	keys4096 := []string{
		`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`,
		`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`,
		`-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`,
	}

	var keyStr string
	switch keyLen {
	case 2048:
		if id >= len(keys2048) {
			t.Fatalf("test key %d [%d bit] not found", id, keyLen)
		}
		keyStr = keys2048[id]
	case 4096:
		if id >= len(keys4096) {
			t.Fatalf("test key %d [%d bit] not found", id, keyLen)
		}
		keyStr = keys4096[id]
	default:
		t.Fatalf("unsupported key length: %d", keyLen)
	}

	block, _ := pem.Decode([]byte(keyStr))
	if block == nil {
		t.Fatalf("failed to decode test key %d [%d bit]", id, keyLen)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse test key %d [%d bit]: %v", id, keyLen, err)
	}
	return key
}
