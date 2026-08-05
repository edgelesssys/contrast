package execserver

// Request messages are sent from client to server.
//
// The first request on a connection must set Cmd. Subsequent messages feed data to the command's
// stdin. Don't set both Cmd and Stdin.
type Request struct {
	// Cmd is the command to execute, as it would be passed to exec* syscalls.
	//
	// It must have at least one element, the binary to run.
	Cmd []string `json:"cmd,omitempty"`

	// Stdin conveys events for the remote process' stdin.
	Stdin *StreamEvent `json:"stdin,omitempty"`
}

// Response messages are sent from server to client.
//
// Each message has exactly one field set. Once the ExitStatus has been sent, no further messages
// are expected on the channel.
type Response struct {
	// Stdout conveys events for the client's stdout.
	Stdout *StreamEvent `json:"stdout,omitempty"`
	// Stderr conveys events for the client's stderr.
	Stderr *StreamEvent `json:"stderr,omitempty"`
	// ExitStatus conveys the UNIX wait status to the client.
	ExitStatus *ExitStatus `json:"exit_status,omitempty"`
}

// StreamEvent transfers stream events to the peer.
//
// Only one of the fields must be set per message.
// If Data is set, it must be non-empty (at least 1 byte).
// Once EOF or Error have been sent, no further messages are expected for this stream.
type StreamEvent struct {
	Data  []byte `json:"data,omitempty"`
	EOF   bool   `json:"eof,omitempty"`
	Error string `json:"error,omitempty"`
}

// ExitStatus models the server process' UNIX wait status.
//
// Exactly one of the fields is set. If the process was signaled, Signaled contains the causing
// signal. If the process terminated normally, Terminated contains the exit code. If something else
// happened, Error is set to a descriptive error message.
type ExitStatus struct {
	Terminated *int   `json:"terminated,omitempty"`
	Signaled   *int   `json:"signaled,omitempty"`
	Error      string `json:"unknown,omitempty"`
}
