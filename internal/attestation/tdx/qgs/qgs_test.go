package qgs

import (
	_ "embed"
	"encoding"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/getCollateralRequest.dat
	getCollateralRequest []byte
	//go:embed testdata/getCollateralResponse.dat
	getCollateralResponse []byte
)

func TestGetCollateralRequest(t *testing.T) {
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

func TestGetCollateralResponse(t *testing.T) {
	require := require.New(t)
	resp := &GetCollateralResponse{}
	require.NoError(resp.UnmarshalBinary(getCollateralResponse))
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
