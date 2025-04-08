// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package peerdiscovery

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Discovery struct {
	client    kubernetes.Interface
	namespace string
}

func New(client kubernetes.Interface, namespace string) *Discovery {
	return &Discovery{
		client:    client,
		namespace: namespace,
	}
}

// GetPeers returns a list of Coordinator IPs that are ready to be used for peer recovery.
func (d *Discovery) GetPeers(ctx context.Context) ([]string, error) {
	// TODO(burgerdev): this should be an informer with cache.
	pods, err := d.client.CoreV1().Pods(d.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{"app.kubernetes.io/name": "coordinator"}.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("listing coordinator pods: %w", err)
	}
	var peers []string
	for _, pod := range pods.Items {
		if pod.Annotations["contrast.edgeless.systems/pod-role"] != "coordinator" ||
			pod.Name == os.Getenv("HOSTNAME") {
			continue
		}
		if isReady(&pod) {
			peers = append(peers, pod.Status.PodIP)
		}
	}
	return peers, nil
}

func isReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
