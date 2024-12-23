// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var version = "0.0.0-dev"

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: nydus-pull <runtime-handler>")
	}

	nodeName := os.Getenv("NODE_NAME")

	if err := run(context.Background(), os.Args[1], nodeName); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

func run(ctx context.Context, runtimeHandler, nodeName string) error {
	log.Printf("nydus-pull version %s\n", version)

	ctrClient, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return fmt.Errorf("failed to connect to containerd: %w", err)
	}
	defer ctrClient.Close()

	ctx = namespaces.WithNamespace(ctx, "k8s.io")

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("getting k8s config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("creating k8s clientset: %w", err)
	}

	watcher, err := kubeClient.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
	})
	if err != nil {
		return fmt.Errorf("getting watcher: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added, watch.Modified:
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				log.Printf("unexpected object type: %T\n", event.Object)
				continue
			}
			if pod.Spec.RuntimeClassName == nil || *pod.Spec.RuntimeClassName != runtimeHandler {
				continue
			}
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Waiting != nil && status.State.Waiting.Reason == "CreateContainerError" {
					if strings.Contains(status.State.Waiting.Message, "failed to get reader from content store") {
						log.Printf("caught pod %s with image pull error: %s\n", pod.Name, status.State.Waiting.Message)
						log.Printf("pulling image: %s\n", status.Image)
						if _, err := ctrClient.Pull(ctx, status.Image); err != nil {
							log.Printf("failed to pull image %s: %v\n", status.Image, err)
							continue
						}
						log.Printf("successfully pulled image: %s\n", status.Image)
					}
				}
			}
		}
	}
	return fmt.Errorf("watcher unexpectedly closed")
}
