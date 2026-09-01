// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// Package qtest accesses QEMU's emulated devices through the qtest protocol.
package qtest

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Client is a qtest protocol client.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewClient creates a qtest client over conn.
func NewClient(conn net.Conn) *Client {
	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (c *Client) command(ctx context.Context, command string) (string, error) {
	deadline := time.Now().Add(time.Minute)
	if contextDeadline, ok := ctx.Deadline(); ok {
		deadline = contextDeadline
	}
	if err := c.conn.SetDeadline(deadline); err != nil {
		return "", fmt.Errorf("setting qtest deadline: %w", err)
	}
	if _, err := io.WriteString(c.conn, command+"\n"); err != nil {
		return "", fmt.Errorf("sending qtest command %q: %w", command, err)
	}

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading response to qtest command %q: %w", command, err)
	}
	response = strings.TrimSuffix(strings.TrimSuffix(response, "\n"), "\r")
	if response != "OK" && !strings.HasPrefix(response, "OK ") {
		return "", fmt.Errorf("qtest command %q failed: %q", command, response)
	}
	return response, nil
}

func (c *Client) outB(ctx context.Context, port uint16, value byte) error {
	_, err := c.command(ctx, fmt.Sprintf("outb 0x%x 0x%x", port, value))
	return err
}

func (c *Client) outL(ctx context.Context, port uint16, value uint32) error {
	_, err := c.command(ctx, fmt.Sprintf("outl 0x%x 0x%x", port, value))
	return err
}

func (c *Client) writeMemory(ctx context.Context, address uint64, data []byte) error {
	_, err := c.command(ctx, fmt.Sprintf("write 0x%x 0x%x 0x%x", address, len(data), data))
	return err
}

func (c *Client) readMemory(ctx context.Context, address uint64, size uint32) ([]byte, error) {
	response, err := c.command(ctx, fmt.Sprintf("b64read 0x%x 0x%x", address, size))
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(response)
	if len(fields) != 2 {
		return nil, fmt.Errorf("malformed b64read response %q", response)
	}
	data, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return nil, fmt.Errorf("decoding b64read response: %w", err)
	}
	if len(data) != int(size) {
		return nil, fmt.Errorf("qtest b64read returned %d bytes, want %d", len(data), size)
	}
	return data, nil
}
