package qgs

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

)

type Client struct {
	conn net.Conn
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) Close() error {
	conn := c.conn
	c.conn = nil
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (c *Client) GetCollateral(ctx context.Context, req *GetCollateralRequest) (*GetCollateralResponse, error) {
	binaryReq, err := req.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshalling %T: %w", req, err)
	}

	var writeErr error
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		if err := binary.Write(c.conn, binary.BigEndian, uint32(len(binaryReq))); err != nil {
			writeErr = fmt.Errorf("writing request length: %w", err)
			return
		}
		if _, err := io.Copy(c.conn, bytes.NewBuffer(binaryReq)); err != nil {
			writeErr = fmt.Errorf("writing request (%d bytes): %w", len(binaryReq), err)
		}
	}()

	select {
	case <-writeDone:
		if writeErr != nil {
			return nil, fmt.Errorf("writing request: %w", err)
		}
	case <-ctx.Done():
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
			return nil, fmt.Errorf("reading response: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("context expired while reading response: %w", ctx.Err())
	}

	resp := &GetCollateralResponse{}
	if err := resp.UnmarshalBinary(binaryResp); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return resp, nil
}
