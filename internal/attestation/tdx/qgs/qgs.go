// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// package qgs provides data structures for interacting with the Intel Quote Generation Service.
//
// There is no publicly documented API, this library is written based on the C++ implementation in
// Intel DCAP.
//
// https://github.com/intel/confidential-computing.tee.dcap/tree/a6c3631/QuoteGeneration/quote_wrapper/qgs_msg_lib
package qgs

import (
	"bytes"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"slices"

	"github.com/google/go-tdx-guest/verify"
)

type messageType = uint32

const (
	messageTypeGetCollateralRequest  messageType = 2
	messageTypeGetCollateralResponse messageType = 3
)

type header struct {
	majorVersion uint16
	minorVersion uint16
	messageType  messageType
	size         uint32
	responseCode uint32
}

const (
	lenHeader = 16
	lenFSMPC  = 6
)

func (h *header) marshalBinary() []byte {
	buf := make([]byte, 0, lenHeader)
	buf = binary.LittleEndian.AppendUint16(buf, h.majorVersion)
	buf = binary.LittleEndian.AppendUint16(buf, h.minorVersion)
	buf = binary.LittleEndian.AppendUint32(buf, h.messageType)
	buf = binary.LittleEndian.AppendUint32(buf, h.size)
	buf = binary.LittleEndian.AppendUint32(buf, h.responseCode)
	return buf
}

func (h *header) unmarshalBinary(data []byte) error {
	if len(data) != lenHeader {
		return fmt.Errorf("wrong header size: expected %d, got %d", lenHeader, len(data))
	}
	h.majorVersion = binary.LittleEndian.Uint16(data[0:2])
	h.minorVersion = binary.LittleEndian.Uint16(data[2:4])
	h.messageType = binary.LittleEndian.Uint32(data[4:8])
	h.size = binary.LittleEndian.Uint32(data[8:12])
	h.responseCode = binary.LittleEndian.Uint32(data[12:16])
	return nil
}

// CAType represents the CA types for which collateral can be fetched.
type CAType string

// Valid CAType values.
const (
	CATypePlatform  CAType = "platform"
	CATypeProcessor CAType = "processor"
)

// GetCollateralRequest asks the server for collateral for the given FMSPC and CAType.
type GetCollateralRequest struct {
	FMSPC  [lenFSMPC]byte
	CAType CAType
}

func (r *GetCollateralRequest) marshalBinary() []byte {
	size := uint32( /*sizes*/ 8 + lenFSMPC + len(r.CAType))
	buf := make([]byte, 0, size)

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(r.FMSPC)))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(r.CAType)))
	buf = append(buf, r.FMSPC[:]...)
	buf = append(buf, r.CAType...)
	return buf
}

// GetCollateralResponse is the QGS response to GetCollateralRequest.
type GetCollateralResponse struct {
	MajorVersion uint16
	MinorVersion uint16

	PCKCRLIssuerChain     []byte
	RootCACRL             []byte
	PCKCRL                []byte
	TCBInfoIssuerChain    []byte
	TCBInfo               []byte
	QEIdentityIssuerChain []byte
	QEIdentity            []byte
}

func (r *GetCollateralResponse) unmarshalBinary(data []byte) error {
	// The response is structured like this:
	//   - Major and Minor version, 2 byte each
	//   - the sizes of all seven field items, 4 byte each
	//   - the field items themselves, variable size
	//
	// In order to simplify the deserialization, we'll treat the field items as array entries we
	// can loop over.
	receivers := []*[]byte{
		&r.PCKCRLIssuerChain,
		&r.RootCACRL,
		&r.PCKCRL,
		&r.TCBInfoIssuerChain,
		&r.TCBInfo,
		&r.QEIdentityIssuerChain,
		&r.QEIdentity,
	}

	// Even if all data fields are empty, we need to have at least the following number of bytes.
	fixedDataLen := /*versions*/ 4 + /*blob lengths*/ 4*len(receivers)
	if len(data) < fixedDataLen {
		return fmt.Errorf("body too short: expected at least %d more bytes, got %d", fixedDataLen, len(data))
	}

	// Two version numbers
	r.MajorVersion = binary.LittleEndian.Uint16(data[0:2])
	r.MinorVersion = binary.LittleEndian.Uint16(data[2:4])

	// We're iterating over both the size headers and the field items simultaneously.
	// i tracks the current field item number, offset tracks the field item positions in the data
	// section.
	offset := uint32(fixedDataLen)
	for i, recv := range receivers {
		size := binary.LittleEndian.Uint32(data[(i+1)*4 : (i+2)*4])
		*recv = data[offset : offset+size]
		offset += size
	}
	if offset != uint32(len(data)) {
		return fmt.Errorf("found %d trailing bytes", len(data)-int(offset))
	}

	// The following items are serialized C-strings and contain a trailing 0x00 byte.
	// We remove it to facilitate further processing in Go (JSON and Hex encodings).
	null := []byte{0}
	r.TCBInfo = bytes.TrimSuffix(r.TCBInfo, null)
	r.QEIdentity = bytes.TrimSuffix(r.QEIdentity, null)
	r.PCKCRL = bytes.TrimSuffix(r.PCKCRL, null)
	r.RootCACRL = bytes.TrimSuffix(r.RootCACRL, null)

	return nil
}

// ToTDXGuest converts the QGS collateral response to the Collateral structure used by go-tdx-guest.
func (r *GetCollateralResponse) ToTDXGuest() (*verify.Collateral, error) {
	c := &verify.Collateral{}

	pckCRLRoot, pckCRLIntermediate, err := parseCertificateChain(r.PCKCRLIssuerChain)
	if err != nil {
		return nil, fmt.Errorf("parsing PCK CRL issuer chain: %w", err)
	}
	c.PckCrlIssuerRootCertificate = pckCRLRoot
	c.PckCrlIssuerIntermediateCertificate = pckCRLIntermediate

	pckCRL, err := parseCRL(r.PCKCRL)
	if err != nil {
		return nil, fmt.Errorf("parsing PCK CRL: %w", err)
	}
	c.PckCrl = pckCRL

	tcbRoot, tcbIntermediate, err := parseCertificateChain(r.TCBInfoIssuerChain)
	if err != nil {
		return nil, fmt.Errorf("parsing TCBInfo issuer chain: %w", err)
	}
	c.TcbInfoIssuerRootCertificate = tcbRoot
	c.TcbInfoIssuerIntermediateCertificate = tcbIntermediate

	if err := json.Unmarshal(r.TCBInfo, &c.TdxTcbInfo); err != nil {
		return nil, fmt.Errorf("parsing TCBInfo: %w", err)
	}

	var tcbInfo map[string]json.RawMessage
	if err := json.Unmarshal(r.TCBInfo, &tcbInfo); err != nil {
		return nil, fmt.Errorf("parsing unstructured TCBInfo: %w", err)
	}
	c.TcbInfoBody = tcbInfo["tcbInfo"]

	qeRoot, qeIntermediate, err := parseCertificateChain(r.QEIdentityIssuerChain)
	if err != nil {
		return nil, fmt.Errorf("parsing QEIdentity issuer chain: %w", err)
	}
	c.QeIdentityIssuerRootCertificate = qeRoot
	c.QeIdentityIssuerIntermediateCertificate = qeIntermediate

	if err := json.Unmarshal(r.QEIdentity, &c.QeIdentity); err != nil {
		return nil, fmt.Errorf("parsing QEIdentity: %w", err)
	}

	var qeIdentity map[string]json.RawMessage
	if err := json.Unmarshal(r.QEIdentity, &qeIdentity); err != nil {
		return nil, fmt.Errorf("parsing unstructured QEIdentity: %w", err)
	}
	c.EnclaveIdentityBody = qeIdentity["enclaveIdentity"]

	rootCRL, err := parseCRL(r.RootCACRL)
	if err != nil {
		return nil, fmt.Errorf("parsing root CA CRL: %w", err)
	}
	c.RootCaCrl = rootCRL

	return c, nil
}

// parseCertificateChain parses concatenated PEM-encoded certificates as they appear in QGS collateral.
//
// This function assumes the following:
//
//   - There are exactly two PEM-encoded certificates in the input.
//   - One of the certificates (i.e. root) signs the other one (i.e. intermediate).
//
// This is in line with the go-tdx-guest implementation, but does not make ordering assumptions:
// https://github.com/google/go-tdx-guest/blob/32866d7/verify/verify.go#L305
func parseCertificateChain(pemChain []byte) (root, intermediate *x509.Certificate, retErr error) {
	certs := []*x509.Certificate{}

	var block *pem.Block
	for len(pemChain) > 0 {
		block, pemChain = pem.Decode(pemChain)
		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing certificate %d: %w", len(certs), err)
		}
		certs = append(certs, cert)
	}
	if len(certs) != 2 {
		return nil, nil, fmt.Errorf("unexpected certificate chain length %d, want 2", len(certs))
	}
	// Order root before intermediate.
	if certs[1].CheckSignatureFrom(certs[0]) != nil {
		slices.Reverse(certs)
	}

	return certs[0], certs[1], nil
}

func parseCRL(hexCRL []byte) (*x509.RevocationList, error) {
	if len(hexCRL) == 0 {
		return nil, nil
	}
	x, err := hex.DecodeString(string(hexCRL))
	if err != nil {
		return nil, fmt.Errorf("hex-decoding CRL: %w", err)
	}
	crl, err := x509.ParseRevocationList(x)
	if err != nil {
		return nil, fmt.Errorf("parsing CRL: %w", err)
	}
	return crl, nil
}
