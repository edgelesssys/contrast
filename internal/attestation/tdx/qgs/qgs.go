package qgs

import (
	"encoding/binary"
	"fmt"
)

type MessageType = uint32

const (
	MessageTypeGetCollateralRequest  MessageType = 2
	MessageTypeGetCollateralResponse MessageType = 3
)

type Header struct {
	MajorVersion uint16
	MinorVersion uint16
	// Ignored when marshalling.
	MessageType MessageType
	// Ignored when marshalling.
	Size         uint32
	ResponseCode uint32
}

const (
	lenHeader = 16
	lenFSMPC  = 6
)

func (h *Header) AppendBinary(buf []byte) ([]byte, error) {
	buf = binary.LittleEndian.AppendUint16(buf, h.MajorVersion)
	buf = binary.LittleEndian.AppendUint16(buf, h.MinorVersion)
	buf = binary.LittleEndian.AppendUint32(buf, h.MessageType)
	buf = binary.LittleEndian.AppendUint32(buf, h.Size)
	buf = binary.LittleEndian.AppendUint32(buf, h.ResponseCode)
	return buf, nil
}

func (h *Header) MarshalBinary() ([]byte, error) {
	return h.AppendBinary(make([]byte, 0, lenHeader))
}

func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) != lenHeader {
		return fmt.Errorf("wrong header size: expected %d, got %d", lenHeader, len(data))
	}
	h.MajorVersion = binary.LittleEndian.Uint16(data[0:2])
	h.MinorVersion = binary.LittleEndian.Uint16(data[2:4])
	h.MessageType = binary.LittleEndian.Uint32(data[4:8])
	h.Size = binary.LittleEndian.Uint32(data[8:12])
	h.ResponseCode = binary.LittleEndian.Uint32(data[12:16])
	return nil
}

type CAType string

const (
	CATypePlatform  CAType = "platform"
	CATypeProcessor CAType = "processor"
)

type GetCollateralRequest struct {
	Header

	FMSPC  [lenFSMPC]byte
	CAType CAType
}

func (r *GetCollateralRequest) MarshalBinary() (data []byte, err error) {
	return r.AppendBinary(make([]byte, 0, r.size()))
}

func (r *GetCollateralRequest) AppendBinary(buf []byte) ([]byte, error) {
	header := r.Header
	header.MessageType = MessageTypeGetCollateralRequest
	header.ResponseCode = 0
	header.Size = r.size()

	var err error
	buf, err = header.AppendBinary(buf)
	if err != nil {
		return nil, err
	}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(r.FMSPC)))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(r.CAType)))
	buf = append(buf, r.FMSPC[:]...)
	buf = append(buf, r.CAType...)
	return buf, nil
}

func (r *GetCollateralRequest) UnmarshalBinary(data []byte) error {
	if err := r.Header.UnmarshalBinary(data[:lenHeader]); err != nil {
		return err
	}
	// TODO(burgerdev): verify message type
	data = data[lenHeader:]
	if len(data) < 8 {
		return fmt.Errorf("body too short: expected at least 8 bytes, got %d", len(data))
	}
	m := binary.LittleEndian.Uint32(data[0:4])
	n := binary.LittleEndian.Uint32(data[4:8])
	data = data[8:]
	// TODO(burgerdev): type issue
	if len(data) != int(m+n) {
		return fmt.Errorf("expected %d+%d bytes in body, got %d", m, n, len(data))
	}
	// TODO(burgerdev): check FMSPC size
	copy(r.FMSPC[:], data[0:m])
	// TODO(burgerdev): check CAType
	r.CAType = CAType(data[m : m+n])
	return nil
}

func (r *GetCollateralRequest) size() uint32 {
	return uint32( /*header*/ 16 + /* sizes*/ 8 + /*fmspc*/ 6 + len(r.CAType))
}

type GetCollateralResponse struct {
	Header `json:"-"`

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

func (r *GetCollateralResponse) UnmarshalBinary(data []byte) error {
	if err := r.Header.UnmarshalBinary(data[:lenHeader]); err != nil {
		return err
	}
	data = data[lenHeader:]

	receivers := []*[]byte{
		&r.PCKCRLIssuerChain,
		&r.RootCACRL,
		&r.PCKCRL,
		&r.TCBInfoIssuerChain,
		&r.TCBInfo,
		&r.QEIdentityIssuerChain,
		&r.QEIdentity,
	}

	fixedDataLen := /*versions*/ 4 + /*blobs*/ 4*len(receivers)
	if len(data) < fixedDataLen {
		return fmt.Errorf("body too short: expected at least %d more bytes, got %d", fixedDataLen, len(data))
	}

	r.MajorVersion = binary.LittleEndian.Uint16(data[0:2])
	r.MinorVersion = binary.LittleEndian.Uint16(data[2:4])

	offset := uint32(fixedDataLen)
	for i, recv := range receivers {
		size := binary.LittleEndian.Uint32(data[(i+1)*4 : (i+2)*4])
		*recv = data[offset : offset+size]
		offset += size
	}
	if offset != uint32(len(data)) {
		return fmt.Errorf("found %d trailing bytes", len(data)-int(offset))
	}

	// TCBInfo is a JSON C-string and includes a trailing 0x00 byte - let's strip it.
	if len(r.TCBInfo) > 0 {
		r.TCBInfo = r.TCBInfo[:len(r.TCBInfo)-1]
	}

	return nil
}
