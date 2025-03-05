// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/seedengine"
	"github.com/edgelesssys/contrast/internal/ca"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func TestNewMeshCert(t *testing.T) {
	m := manifestWithWorkloadOwnerKey(t, testkeys.ECDSA(t))
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
	require.NoError(t, err)
	ca, err := ca.New(rootKey, meshKey)
	require.NoError(t, err)

	info := AuthInfo{
		TLSInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{{PublicKey: key.Public(), PublicKeyAlgorithm: x509.ECDSA}},
			},
		},
		Report: &fakeReport{
			hostData: policyHash[:],
		},
		State: &State{
			Manifest:   m,
			SeedEngine: se,
			CA:         ca,
		},
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: info,
	})

	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(store)
	authority := New(hist, prometheus.NewRegistry(), slog.Default())

	resp, err := authority.NewMeshCert(ctx, nil)
	require.NoError(t, err)

	require.NotEmpty(t, resp.WorkloadSecret)

	certChain := certFromPEM(t, resp.CertChain)
	cert, intermediateCert := certChain[0], certChain[1]
	assert.False(t, cert.IsCA)
	assert.True(t, intermediateCert.IsCA)
	assert.Equal(t, cert.AuthorityKeyId, intermediateCert.SubjectKeyId)

	meshCert := certFromPEM(t, resp.MeshCACert)[0]
	assert.True(t, meshCert.IsCA)
	assert.Equal(t, cert.AuthorityKeyId, meshCert.SubjectKeyId)
	assert.Empty(t, meshCert.AuthorityKeyId)

	rootCert := certFromPEM(t, resp.RootCACert)[0]
	assert.True(t, rootCert.IsCA)
	assert.Empty(t, rootCert.AuthorityKeyId)
	assert.Equal(t, intermediateCert.AuthorityKeyId, rootCert.SubjectKeyId)
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
