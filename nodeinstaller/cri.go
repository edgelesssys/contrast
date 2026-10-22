// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func needsRestartCRI(ctx context.Context, socket, runtimeHandler string) (bool, error) {
	conn, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false, fmt.Errorf("creating CRI client for %q: %w", socket, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := criv1.NewRuntimeServiceClient(conn)
	resp, err := client.Status(ctx, &criv1.StatusRequest{})
	if err != nil {
		return false, fmt.Errorf("calling CRI RuntimeService.Status: %w", err)
	}

	handlerLoaded := slices.ContainsFunc(resp.RuntimeHandlers, func(h *criv1.RuntimeHandler) bool {
		return h != nil && h.Name == runtimeHandler
	})

	return !handlerLoaded, nil
}

// TODO(burgerdev): add more checks on the loaded config
// If you pass `Verbose: true` you will get additional data in the `resp.Info["config"]` field.
// This is an encoded JSON object that can be queried like shown below for more information about
// the loaded runtime handler.

/*
$ cat status.protojson | jq -r '.info.config' | jq '.containerd.runtimes["contrast-cc-metal-qemu-snp-ecd99d77"]'
{
  "runtimeType": "io.containerd.contrast-cc.v2",
  "runtimePath": "/opt/edgeless/contrast-cc-metal-qemu-snp-ecd99d77/bin/containerd-shim-contrast-cc-v2",
  "PodAnnotations": [
    "io.katacontainers.*"
  ],
  "ContainerAnnotations": null,
  "options": {
    "ConfigPath": "/opt/edgeless/contrast-cc-metal-qemu-snp-ecd99d77/etc/configuration-qemu-snp.toml"
  },
  "privileged_without_host_devices": true,
  "privileged_without_host_devices_all_devices_allowed": false,
  "cgroupWritable": false,
  "baseRuntimeSpec": "",
  "cniConfDir": "",
  "cniMaxConfNum": 0,
  "snapshotter": "",
  "sandboxer": "podsandbox",
  "io_type": ""
}
*/
