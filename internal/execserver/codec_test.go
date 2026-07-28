package execserver

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteReadMessageRoundTrip(t *testing.T) {
	require := require.New(t)
	want := &Request{Cmd: []string{"echo", "hi"}}

	var buf bytes.Buffer
	require.NoError(WriteMessage(&buf, want))

	var got Request
	require.NoError(ReadMessage(&buf, &got))
	require.Equal(want, &got)
}

func TestWriteMessageRejectsOversized(t *testing.T) {
	var buf bytes.Buffer
	require.ErrorIs(t, WriteMessage(&buf, &StreamEvent{Data: make([]byte, MaxMessageSize+1)}), ErrMessageTooLarge)
}

func TestReadMessageRejectsOversizedLengthPrefix(t *testing.T) {
	// A length prefix claiming more than MaxMessageSize must be rejected before
	// ReadMessage allocates a buffer for the (nonexistent) payload.
	var got StreamEvent
	require.ErrorIs(t, ReadMessage(bytes.NewReader([]byte{0xFF, 0xFF, 0xFF, 0xFF}), &got), ErrMessageTooLarge)
}
