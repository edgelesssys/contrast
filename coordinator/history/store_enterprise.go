// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package history

import (
	"log/slog"
	"os"

	"github.com/edgelesssys/contrast/enterprise/coordinator/history"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewStore creates a new ConfigMapStore backed by Kubernetes Config Maps.
func NewStore(log *slog.Logger) (*history.ConfigMapStore, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, err
	}
	return history.NewConfigMapStore(clientset, string(namespace), log)
}
