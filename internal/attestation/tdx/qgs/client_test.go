// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qgs

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"
)

func TestClient(t *testing.T) {
	require := require.New(t)

	c, s := net.Pipe()
	t.Cleanup(func() { _ = s.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	spy := spyConn{
		conn:     s,
		response: getCollateralResponse,
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return spy.Run(ctx)
	})

	client := NewClient(c)
	t.Cleanup(func() { _ = client.Close() })

	req := &GetCollateralRequest{
		FMSPC:  [lenFSMPC]byte{0x90, 0xc0, 0x6f, 0x00, 0x00, 0x00},
		CAType: CATypePlatform,
	}

	resp, err := client.GetCollateral(ctx, req)
	require.NoError(err)
	require.NotNil(resp)
	require.NoError(eg.Wait())

	require.Equal(getCollateralRequest, spy.observedRequest)
}

func TestClient_OversizedResponse(t *testing.T) {
	require := require.New(t)

	c, s := net.Pipe()
	t.Cleanup(func() { _ = s.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	// Announce a response of maximal size, but never send it.
	spy := spyConn{
		conn:           s,
		responseLength: math.MaxUint32,
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return spy.Run(ctx)
	})

	client := NewClient(c)
	t.Cleanup(func() { _ = client.Close() })

	req := &GetCollateralRequest{
		FMSPC:  [lenFSMPC]byte{0x90, 0xc0, 0x6f, 0x00, 0x00, 0x00},
		CAType: CATypePlatform,
	}

	_, err := client.GetCollateral(ctx, req)
	require.ErrorContains(err, "exceeds maximum")
	require.NoError(eg.Wait())
}

func TestClient_UnresponsiveServer(t *testing.T) {
	require := require.New(t)

	c, s := net.Pipe()
	t.Cleanup(func() { _ = s.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	client := NewClient(c)
	t.Cleanup(func() { _ = client.Close() })

	req := &GetCollateralRequest{
		FMSPC:  [lenFSMPC]byte{0x90, 0xc0, 0x6f, 0x00, 0x00, 0x00},
		CAType: CATypePlatform,
	}

	_, err := client.GetCollateral(ctx, req)
	require.ErrorIs(err, context.Canceled)
}

type spyConn struct {
	// conn is connected to the client under test
	conn net.Conn
	// response will be sent, prefixed by the big-endian size header
	response []byte
	// responseLength overrides the size header, if set.
	responseLength uint32
	// observedRequest will be filled with the serialized request object.
	observedRequest []byte
}

func (c *spyConn) Run(context.Context) error {
	var size uint32
	if err := binary.Read(c.conn, binary.BigEndian, &size); err != nil {
		return err
	}
	c.observedRequest = make([]byte, size)
	if _, err := io.ReadFull(c.conn, c.observedRequest); err != nil {
		return err
	}

	length := c.responseLength
	if length == 0 {
		length = uint32(len(c.response))
	}
	if err := binary.Write(c.conn, binary.BigEndian, length); err != nil {
		return err
	}
	if _, err := io.Copy(c.conn, bytes.NewBuffer(c.response)); err != nil {
		return err
	}

	return nil
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
