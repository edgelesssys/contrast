// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package qtest

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"testing"
)

func TestReadMemory(t *testing.T) {
	const testAddress = 0x08000000

	clientConnection, serverConnection := net.Pipe()
	defer clientConnection.Close()
	defer serverConnection.Close()

	want := []byte{0, 1, 2, 0xfe, 0xff}
	serverError := make(chan error, 1)
	go func() {
		request, err := bufio.NewReader(serverConnection).ReadString('\n')
		if err != nil {
			serverError <- err
			return
		}
		if request != "b64read 0x8000000 0x5\n" {
			serverError <- fmt.Errorf("request = %q", request)
			return
		}
		_, err = fmt.Fprintf(serverConnection, "OK %s\n", base64.StdEncoding.EncodeToString(want))
		serverError <- err
	}()

	got, err := NewClient(context.Background(), clientConnection).readMemory(testAddress, uint32(len(want)))
	if err != nil {
		t.Fatalf("readMemory: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readMemory() = %x, want %x", got, want)
	}
	if err := <-serverError; err != nil {
		t.Fatalf("fake qtest server: %v", err)
	}
}

func TestRejectsErrorResponse(t *testing.T) {
	clientConnection, serverConnection := net.Pipe()
	defer clientConnection.Close()
	defer serverConnection.Close()

	go func() {
		_, _ = bufio.NewReader(serverConnection).ReadString('\n')
		_, _ = fmt.Fprintln(serverConnection, "FAIL unsupported command")
	}()
	if _, err := NewClient(context.Background(), clientConnection).command("bad-command"); err == nil {
		t.Fatal("command returned nil error")
	}
}
