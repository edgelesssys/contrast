package qgs

import (
	_ "embed"
	"encoding"
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
	req := &GetCollateralRequest{}
	require.NoError(req.UnmarshalBinary(getCollateralRequest))

	expectedReq := &GetCollateralRequest{
		Header: Header{
			MajorVersion: 0x01,
			MinorVersion: 0x01,
			MessageType:  0x02,
			Size:         0x26,
			ResponseCode: 0x00,
		},
		FMSPC:  [lenFSMPC]byte{0x90, 0xc0, 0x6f, 0x00, 0x00, 0x00},
		CAType: CATypePlatform,
	}
	require.Equal(expectedReq, req)
	actual, err := req.MarshalBinary()
	require.NoError(err)
	require.Equal(getCollateralRequest, actual)
}

func TestToTDXGuest(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	resp := &GetCollateralResponse{}
	require.NoError(resp.UnmarshalBinary(getCollateralResponse))

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

// Test interface implementations.
var (
	_ = encoding.BinaryAppender(&Header{})
	_ = encoding.BinaryMarshaler(&Header{})
	_ = encoding.BinaryAppender(&GetCollateralRequest{})
	_ = encoding.BinaryMarshaler(&GetCollateralRequest{})
	_ = encoding.BinaryUnmarshaler(&GetCollateralRequest{})
	_ = encoding.BinaryUnmarshaler(&GetCollateralResponse{})
)
