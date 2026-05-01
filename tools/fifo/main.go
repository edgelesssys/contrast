// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// fifo is a CLI tool for FIFO-ordered mutual exclusion via Kubernetes Lease objects.
// Callers acquire a named lock and receive a holder identity they later use to release it.
// The queue is modeled as Lease objects, too. See the internal/lease package for details.
package main

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	coordinationtypesv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultLeaseName = "gpu-sync"
	defaultNamespace = "default"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "fifo",
		Short:        "FIFO lock backed by Kubernetes Lease objects",
		SilenceUsage: true,
	}
	root.AddCommand(newAcquireCmd())
	root.AddCommand(newReleaseCmd())
	return root
}

func newLeaseClient(namespace string) (coordinationtypesv1.LeaseInterface, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return client.CoordinationV1().Leases(namespace), nil
}
