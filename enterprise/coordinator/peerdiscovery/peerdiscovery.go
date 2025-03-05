// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build enterprise

package peerdiscovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// PeerStore is a store of Coordinators that are ready to be used for peer recovery.
type PeerStore struct {
	peers map[string]struct{}
	mu    sync.RWMutex

	client          kubernetes.Interface
	namespace       string
	hostname        string
	informerFactory informers.SharedInformerFactory
	logger          *slog.Logger
}

// New creates a new PeerStore that watches for Coordinator pods in the Kubernetes cluster.
func New(logger *slog.Logger) (*PeerStore, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	factory := informers.NewSharedInformerFactoryWithOptions(
		client,
		0,
		informers.WithNamespace(string(namespace)),
		informers.WithTweakListOptions(
			func(options *metav1.ListOptions) {
				options.LabelSelector = labels.Set{"app.kubernetes.io/name": "coordinator"}.String()
			},
		),
	)
	return &PeerStore{
		peers:           make(map[string]struct{}),
		client:          client,
		informerFactory: factory,
		namespace:       string(namespace),
		hostname:        hostname,
		logger:          logger.WithGroup("peer-discovery"),
	}, nil
}

// Run starts watching for Coordinator peers and poopulates a local
// store with ready Coordinators to use for peer recovery.
func (p *PeerStore) Run(ctx context.Context) error {
	informer := p.informerFactory.Core().V1().Pods().Informer()
	_, err := informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    p.updatePeers,
			UpdateFunc: func(_, obj interface{}) { p.updatePeers(obj) },
			DeleteFunc: p.updatePeers,
		})
	if err != nil {
		return fmt.Errorf("adding event handler: %w", err)
	}

	p.logger.Info("Starting peer discovery")
	p.informerFactory.Start(ctx.Done())

	<-ctx.Done()
	return nil
}

// GetPeers returns a list of Coordinator IPs that are ready to be used for peer recovery.
func (p *PeerStore) GetPeers() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	peers := make([]string, len(p.peers))
	for peer := range p.peers {
		peers = append(peers, peer)
	}
	return peers
}

// Stop stops the informer factory.
func (p *PeerStore) Stop() {
	p.informerFactory.Shutdown()
}

func (p *PeerStore) updatePeers(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		p.logger.Warn("unexpected object type", "type", fmt.Sprintf("%T", obj))
		return
	}
	if pod.ObjectMeta.Annotations["contrast.edgeless.systems/pod-role"] != "coordinator" ||
		pod.ObjectMeta.Name == p.hostname {
		return
	}
	p.mu.Lock()
	beforeLen := len(p.peers)
	if isReady(pod) {
		p.peers[pod.Status.PodIP] = struct{}{}
	} else {
		delete(p.peers, pod.Status.PodIP)
	}
	afterLen := len(p.peers)
	p.mu.Unlock()
	if beforeLen != afterLen {
		p.logger.Debug("updated peers", "peers", p.GetPeers())
	}
}

func isReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
