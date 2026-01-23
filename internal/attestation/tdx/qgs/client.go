// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qgs

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// Client facilitates RPCs with the Intel QGS.
//
// Create instances with NewClient.
type Client struct {
	conn net.Conn
}

// NewClient creates a new Client instance for the provided connected socket.
//
// The QGS server is usually listening on VSOCK CID 2 (host) port 4050.
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

// Close closes this client and the underlying network connection.
func (c *Client) Close() error {
	conn := c.conn
	c.conn = nil
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// GetCollateral runs the GetCollateral RPC.
//
// If GetCollateral returns an error, it's very likely that the underlying connection is in a
// broken state and should be closed and recreated.
func (c *Client) GetCollateral(ctx context.Context, req *GetCollateralRequest) (*GetCollateralResponse, error) {
	binaryReq := req.marshalBinary()

	reqHeader := &header{
		majorVersion: 1,
		minorVersion: 1,
		messageType:  messageTypeGetCollateralRequest,
		responseCode: 0,
		size:         uint32(len(binaryReq) + lenHeader),
	}

	// We're writing from a Goroutine so that we catch context cancellations.

	var writeErr error
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		if err := binary.Write(c.conn, binary.BigEndian, reqHeader.size); err != nil {
			writeErr = fmt.Errorf("writing request length: %w", err)
			return
		}
		binaryHeader := reqHeader.marshalBinary()
		if _, err := io.Copy(c.conn, bytes.NewBuffer(binaryHeader)); err != nil {
			writeErr = fmt.Errorf("writing request header: %w", err)
		}
		if _, err := io.Copy(c.conn, bytes.NewBuffer(binaryReq)); err != nil {
			writeErr = fmt.Errorf("writing request (%d bytes): %w", len(binaryReq), err)
		}
	}()

	select {
	case <-writeDone:
		if writeErr != nil {
			return nil, fmt.Errorf("writing request: %w", writeErr)
		}
	case <-ctx.Done():
		// Set deadline so that the Goroutine stops.
		if c.conn.SetDeadline(time.Now()) == nil {
			<-writeDone
		}
		return nil, fmt.Errorf("context expired while writing request: %w", ctx.Err())
	}

	var readErr error
	readDone := make(chan struct{})
	var binaryResp []byte
	go func() {
		defer close(readDone)
		var length uint32
		if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
			readErr = fmt.Errorf("reading response length: %w", err)
			return
		}

		binaryResp = make([]byte, length)
		if _, err := io.ReadFull(c.conn, binaryResp); err != nil {
			readErr = fmt.Errorf("reading response (%d byte): %w", length, err)
		}
	}()

	select {
	case <-readDone:
		if readErr != nil {
			return nil, fmt.Errorf("reading response: %w", readErr)
		}
	case <-ctx.Done():
		// Set deadline so that the Goroutine stops.
		if c.conn.SetDeadline(time.Now()) == nil {
			<-readDone
		}
		return nil, fmt.Errorf("context expired while reading response: %w", ctx.Err())
	}

	if len(binaryResp) < lenHeader {
		return nil, fmt.Errorf("response too short for header: got %d bytes, want %d", len(binaryResp), lenHeader)
	}
	var respHeader header
	if err := respHeader.unmarshalBinary(binaryResp[:lenHeader]); err != nil {
		return nil, fmt.Errorf("unmarshallinnng header: %w", err)
	}

	if respHeader.majorVersion != 1 || respHeader.minorVersion != 1 {
		return nil, fmt.Errorf("unsupported QGS protocol version %d.%d, expect 1.1", respHeader.majorVersion, respHeader.minorVersion)
	}

	if respHeader.responseCode != 0 {
		return nil, fmt.Errorf("QGS returned an error response: %d", respHeader.responseCode)
	}

	if respHeader.messageType != messageTypeGetCollateralResponse {
		return nil, fmt.Errorf("unexpected message type %d, expect %d", respHeader.messageType, messageTypeGetCollateralResponse)
	}

	resp := &GetCollateralResponse{}
	if err := resp.unmarshalBinary(binaryResp[lenHeader:]); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return resp, nil
}
