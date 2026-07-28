package execserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxMessageSize is the largest JSON message accepted by ReadMessage.
const MaxMessageSize = 1 << 20 // 1 MiB

const readBufSize = 32 * 1024

// WriteMessage marshals v to JSON and writes it to w as a single frame, prefixed with its
// length as a 4-byte big-endian integer.
func WriteMessage(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}
	if len(data) > MaxMessageSize {
		return fmt.Errorf("%w: maximum %d bytes, encoded message is %d bytes", ErrMessageTooLarge, MaxMessageSize, len(data))
	}

	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	return nil
}

// ReadMessage reads a single length-prefixed frame from r, as written by WriteMessage, and
// unmarshals its JSON payload into v.
func ReadMessage(r io.Reader, v any) error {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}
	if size > MaxMessageSize {
		return fmt.Errorf("%w: maximum %d bytes, incoming message is %d bytes", ErrMessageTooLarge, MaxMessageSize, size)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return fmt.Errorf("reading message: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshaling message: %w", err)
	}
	return nil
}

// ErrMessageTooLarge is returned if the serialized message exceeds the maximum size.
var ErrMessageTooLarge = errors.New("message to large")
