// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/edgelesssys/contrast/internal/imagepuller"

	"github.com/containerd/ttrpc"
)

func main() {
	conn, err := net.Dial("unix", imagepuller.Socket)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := ttrpc.NewClient(conn)
	defer client.Close()

	imagePullerClient := imagepuller.NewImagePullerClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := imagePullerClient.PullImage(ctx, &imagepuller.ImagePullRequest{ImageUrl: "docker.io/library/alpine:latest", BundlePath: "here"})
	if err != nil {
		log.Fatalf("RPC failed: %v", err)
	}

	log.Printf("Client got response: %s", resp)
}
