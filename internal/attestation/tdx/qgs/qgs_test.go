// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qgs

import (
	_ "embed"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/getCollateralRequest.dat
	getCollateralRequest []byte
	//go:embed testdata/getCollateralResponse.dat
	getCollateralResponse []byte
)

func TestRequestMarshalling(t *testing.T) {
	require := require.New(t)

	req := &GetCollateralRequest{
		FMSPC:  [lenFSMPC]byte{0x90, 0xc0, 0x6f, 0x00, 0x00, 0x00},
		CAType: CATypePlatform,
	}
	binaryReq := req.marshalBinary()

	header := header{
		majorVersion: 1,
		minorVersion: 1,
		messageType:  messageTypeGetCollateralRequest,
		responseCode: 0,
		size:         uint32(len(binaryReq) + lenHeader),
	}

	require.Equal(getCollateralRequest, append(header.marshalBinary(), binaryReq...))
}

func TestResponseMarshalling(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var body GetCollateralResponse
	require.NoError(body.unmarshalBinary(getCollateralResponse[lenHeader:]))
	assert.NotNil(body.PCKCRL)
	assert.NotNil(body.PCKCRLIssuerChain)
	assert.NotNil(body.TCBInfo)
	assert.NotNil(body.TCBInfoIssuerChain)
	assert.NotNil(body.QEIdentity)
	assert.NotNil(body.QEIdentityIssuerChain)
	assert.NotNil(body.RootCACRL)
}

func TestToTDXGuest(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	resp := &GetCollateralResponse{}
	require.NoError(resp.unmarshalBinary(getCollateralResponse[lenHeader:]))

	collateral, err := resp.ToTDXGuest()
	require.NoError(err)

	assert.NotNil(collateral.PckCrlIssuerIntermediateCertificate)
	assert.NotNil(collateral.PckCrlIssuerRootCertificate)
	assert.NotNil(collateral.PckCrl)
	assert.NotNil(collateral.TcbInfoIssuerIntermediateCertificate)
	assert.NotNil(collateral.TcbInfoIssuerRootCertificate)
	assert.NotZero(collateral.TdxTcbInfo)
	assert.NotNil(collateral.TcbInfoBody)
	assert.NotNil(collateral.QeIdentityIssuerIntermediateCertificate)
	assert.NotNil(collateral.QeIdentityIssuerRootCertificate)
	assert.NotZero(collateral.QeIdentity)
	assert.NotNil(collateral.EnclaveIdentityBody)
	assert.NotNil(collateral.RootCaCrl)
}

func TestResponseUnmarshalling_Malformed(t *testing.T) {
	// body builds a response body with the given blob sizes and payload length.
	body := func(payload int, sizes ...uint32) []byte {
		buf := binary.LittleEndian.AppendUint16(nil, 1)
		buf = binary.LittleEndian.AppendUint16(buf, 0)
		for _, s := range sizes {
			buf = binary.LittleEndian.AppendUint32(buf, s)
		}
		return append(buf, make([]byte, payload)...)
	}

	testCases := map[string][]byte{
		"too short":           body(0, 0, 0, 0),
		"blob exceeds body":   body(4, 0, 0, 0, 0, 0, 0, 5),
		"blob sizes overflow": body(4, 4, math.MaxUint32, 0, 0, 0, 0, 0),
		"trailing bytes":      body(4, 0, 0, 0, 0, 0, 0, 3),
	}

	for name, data := range testCases {
		t.Run(name, func(t *testing.T) {
			var resp GetCollateralResponse
			assert.Error(t, resp.unmarshalBinary(data))
		})
	}
}

func FuzzResponseUnmarshalling(f *testing.F) {
	f.Add(getCollateralResponse[lenHeader:])
	f.Fuzz(func(_ *testing.T, data []byte) {
		var resp GetCollateralResponse
		_ = resp.unmarshalBinary(data)
	})
}
