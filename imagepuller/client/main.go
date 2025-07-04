// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/containerd/ttrpc"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
)

func main() {
	image := os.Args[1]
	mount := os.Args[2]

	conn, err := net.Dial("unix", api.Socket)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := ttrpc.NewClient(conn)
	defer client.Close()

	imagePullerClient := api.NewImagePullServiceClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := imagePullerClient.PullImage(ctx, &api.ImagePullRequest{ImageUrl: image, BundlePath: mount})
	if err != nil {
		log.Fatalf("RPC failed: %v", err)
	}

	log.Printf("Client got response: %s", resp)
}
