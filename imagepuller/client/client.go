// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/containerd/ttrpc"
	"github.com/edgelesssys/contrast/internal/katacomponents"
)

// Request makes an imagepulling request to the imagepuller ttrpc server.
func Request(image, mount string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", katacomponents.ImagepullSocket)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := ttrpc.NewClient(conn)
	defer client.Close()

	imagePullerClient := katacomponents.NewImagePullServiceClient(client)

	_, err = imagePullerClient.PullImage(ctx, &katacomponents.ImagePullRequest{ImageUrl: image, BundlePath: mount})
	return err
}
