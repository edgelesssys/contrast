// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package meshapi

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func TestNewMeshCert(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	m := &manifest.Manifest{}
	policyHash := sha256.Sum256(nil)
	policyHashHex := manifest.NewHexString(policyHash[:])
	m.Policies = map[manifest.HexString]manifest.PolicyEntry{
		policyHashHex: {
			SANs:             []string{"test"},
			WorkloadSecretID: "test",
		},
	}
	key := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[0])
	meshKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[1])
	rootKey := testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP384Keys[2])

	seed := [32]byte{}
	salt := [32]byte{}
	se, err := seedengine.New(seed[:], salt[:])
	require.NoError(err)
	ca, err := ca.New(rootKey, meshKey)
	require.NoError(err)

	info := stateguard.AuthInfo{
		TLSInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{{PublicKey: key.Public(), PublicKeyAlgorithm: x509.ECDSA}},
			},
		},
		Report: &fakeReport{
			hostData: policyHash[:],
		},
		State: stateguard.NewStateForTest(se, m, nil, ca),
	}
	ctx := peer.NewContext(t.Context(), &peer.Peer{
		AuthInfo: info,
	})

	meshapi := New(slog.Default())

	resp, err := meshapi.NewMeshCert(ctx, nil)
	require.NoError(err)

	require.NotEmpty(resp.WorkloadSecret)

	certChain := certFromPEM(t, resp.CertChain)
	require.Len(certChain, 2)
	cert, intermediateCert := certChain[0], certChain[1]
	assert.False(cert.IsCA)
	assert.True(intermediateCert.IsCA)
	assert.Equal(cert.AuthorityKeyId, intermediateCert.SubjectKeyId)

	meshCerts := certFromPEM(t, resp.MeshCACert)
	require.Len(meshCerts, 1)
	assert.True(meshCerts[0].IsCA)
	assert.Equal(cert.AuthorityKeyId, meshCerts[0].SubjectKeyId)
	assert.Empty(meshCerts[0].AuthorityKeyId)

	rootCerts := certFromPEM(t, resp.RootCACert)
	require.Len(rootCerts, 1)
	assert.True(rootCerts[0].IsCA)
	assert.Empty(rootCerts[0].AuthorityKeyId)
	assert.Equal(intermediateCert.AuthorityKeyId, rootCerts[0].SubjectKeyId)
}

func TestRecover(t *testing.T) {
	testCases := map[string]struct {
		mnfst   *manifest.Manifest
		report  *fakeReport
		wantErr bool
	}{
		"default": {
			mnfst: &manifest.Manifest{
				Policies: map[manifest.HexString]manifest.PolicyEntry{
					"0000000000000000000000000000000000000000000000000000000000000000": {
						Role: manifest.RoleCoordinator,
					},
				},
			},
			report: &fakeReport{
				hostData: bytes.Repeat([]byte{0}, 32),
			},
		},
		"unknown policy hash": {
			mnfst: &manifest.Manifest{},
			report: &fakeReport{
				hostData: bytes.Repeat([]byte{0}, 32),
			},
			wantErr: true,
		},
		"role not coordinator": {
			mnfst: &manifest.Manifest{
				Policies: map[manifest.HexString]manifest.PolicyEntry{
					"0000000000000000000000000000000000000000000000000000000000000000": {},
				},
			},
			report: &fakeReport{
				hostData: bytes.Repeat([]byte{0}, 32),
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			mJSON, err := json.Marshal(tc.mnfst)
			require.NoError(err)

			seed := [32]byte{}
			salt := [32]byte{}
			se, err := seedengine.New(seed[:], salt[:])
			require.NoError(err)

			meshKey := testkeys.ECDSA(t)
			meshKeyDER, err := x509.MarshalECPrivateKey(meshKey)
			require.NoError(err)
			meshKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: meshKeyDER})

			ca, err := ca.New(se.RootCAKey(), meshKey)
			require.NoError(err)

			info := stateguard.AuthInfo{
				Report: &fakeReport{
					hostData: tc.report.hostData,
				},
				State: stateguard.NewStateForTest(se, tc.mnfst, mJSON, ca),
			}
			ctx := peer.NewContext(t.Context(), &peer.Peer{
				AuthInfo: info,
			})

			meshapi := New(slog.Default())

			resp, err := meshapi.Recover(ctx, nil)
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			assert.Equal(seed[:], resp.Seed)
			assert.Equal(salt[:], resp.Salt)
			assert.Equal(meshKeyPEM, resp.MeshCAKey)
			assert.JSONEq(string(mJSON), string(resp.LatestManifest))
		})
	}
}

type fakeReport struct {
	extensions []pkix.Extension
	hostData   []byte
}

func (r *fakeReport) ClaimsToCertExtension() ([]pkix.Extension, error) {
	return r.extensions, nil
}

func (r *fakeReport) HostData() []byte {
	return r.hostData
}

func certFromPEM(t *testing.T, pemBytes []byte) []*x509.Certificate {
	t.Helper()
	var certs []*x509.Certificate
	for len(pemBytes) > 0 {
		derCert, rest := pem.Decode(pemBytes)
		cert, err := x509.ParseCertificate(derCert.Bytes)
		require.NoError(t, err)
		certs = append(certs, cert)
		pemBytes = rest
	}
	return certs
}
