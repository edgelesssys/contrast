// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package main

import (
	"context"
	"os"

	"github.com/edgelesssys/contrast/enterprise/coordinator/peerdiscovery"
	"github.com/edgelesssys/contrast/enterprise/coordinator/recovery"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"
)

func registerEnterpriseServices(ctx context.Context, eg *errgroup.Group, c *components) {
	eg.Go(func() error {
		c.logger.Info("Watching manifest store")
		if err := c.auth.WatchHistory(ctx); err != nil {
			c.logger.Error("Watching manifest store", "err", err)
		}
		return nil
	})

	eg.Go(func() error {
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return err
		}
		discovery := peerdiscovery.New(clientset, string(namespace))
		return recovery.New(c.auth, discovery, c.hist, c.issuer, c.httpsGetter, c.logger).RunRecovery(ctx, clock.RealClock{})
	})
}
