// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeclient

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// WaitForDeployment waits until the Deployment is ready.
//
// We consider the Deployment ready when the number of ready pods targeted by the Deployment is
// exactly the number of desired replicas. Changes in the desired number of replicas are not taken
// into account while waiting!
func (c *Kubeclient) WaitForDeployment(ctx context.Context, namespace, name string) error {
	d, err := c.Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(d.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: int(*d.Spec.Replicas)})
}

// WaitForStatefulSet waits until the StatefulSet is ready.
//
// We consider the StatefulSet ready when the number of ready pods targeted by the StatefulSet is
// exactly the number of desired replicas. Changes in the desired number of replicas are not taken
// into account while waiting!
func (c *Kubeclient) WaitForStatefulSet(ctx context.Context, namespace, name string) error {
	s, err := c.Client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(s.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: int(*s.Spec.Replicas)})
}

// WaitForCoordinator waits until the first Coordinator is started.
//
// The Coordinator only becomes ready when it has a manifest configured, but for setting a manifest
// we only need it to be running. Since the Coordinator is managed by a StatefulSet, additional
// replicas will only be started after the first Coordinator becomes ready, so we only need to wait
// for the first instance to become running.
func (c *Kubeclient) WaitForCoordinator(ctx context.Context, namespace string) error {
	s, err := c.Client.AppsV1().StatefulSets(namespace).Get(ctx, "coordinator", metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(s.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &oneRunning{ls: ls})
}

// WaitForDaemonSet waits until the DaemonSet is ready.
//
// We consider the DaemonSet ready when the number of ready pods targeted by the DaemonSet is
// exactly the number of nodes in the cluster. Changes in the number of nodes are not taken
// into account while waiting!
func (c *Kubeclient) WaitForDaemonSet(ctx context.Context, namespace, name string) error {
	// TODO(burgerdev): this does not take scheduler considerations, like taints, into account.
	nodes, err := c.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	d, err := c.Client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(d.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: len(nodes.Items)})
}

// WaitForPod waits until the pod is ready.
func (c *Kubeclient) WaitForPod(ctx context.Context, namespace, name string) error {
	return c.WaitForPodCondition(ctx, namespace, &singlePodReady{name: name})
}

// WaitForJob waits until the Job succeeded.
//
// We consider the Job succeeded if the Pod belonging to the Job succeeded.
func (c *Kubeclient) WaitForJob(ctx context.Context, namespace, name string) error {
	j, err := c.Client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(j.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numSucceeded{ls: ls, n: 1})
}

// WaitForReplicaSet waits until the ReplicaSet is ready.
//
// We consider the ReplicaSet ready when the number of ready pods targeted by the ReplicaSet is
// exactly the number of desired replicas. Changes in the desired number of replicas are not taken
// into account while waiting!
func (c *Kubeclient) WaitForReplicaSet(ctx context.Context, namespace, name string) error {
	s, err := c.Client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(s.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: int(*s.Spec.Replicas)})
}

// WaitForReplicationController waits until the ReplicationController is ready.
//
// We consider the ReplicationController ready when the number of ready pods targeted by the ReplicationController is
// exactly the number of desired replicas. Changes in the desired number of replicas are not taken
// into account while waiting!
func (c *Kubeclient) WaitForReplicationController(ctx context.Context, namespace, name string) error {
	s, err := c.Client.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ls := labels.SelectorFromSet(s.Spec.Selector)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: int(*s.Spec.Replicas)})
}

// PodCondition indicates the current status of pods to WaitForPodCondition.
//
// PodConditions are logged frequently and should thus implement fmt.Stringer.
type PodCondition interface {
	// Check is called by WaitForPodCondition whenever the cluster's pods change.
	//
	// It receives a namespaced pod lister and returns true if the desired condition is satisfied,
	// false if waiting should continue, or an error if something went wrong.
	Check(listerscorev1.PodLister) (bool, error)
}

// WaitForPodCondition waits until the pods in the given namespace satisfy the given condition or the context expires.
func (c *Kubeclient) WaitForPodCondition(ctx context.Context, namespace string, podCondition PodCondition) error {
	logger := c.log.With("namespace", namespace)
	factory := informers.NewSharedInformerFactoryWithOptions(c.Client, 5*time.Second, informers.WithNamespace(namespace))

	// The notifications channel will be written to whenever the informer records a pod change.
	notifications := make(chan struct{}, 256)
	registration, err := factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(any) {
			notifications <- struct{}{}
		},
		UpdateFunc: func(any, any) {
			notifications <- struct{}{}
		},
		DeleteFunc: func(any) {
			notifications <- struct{}{}
		},
	})
	if err != nil {
		return fmt.Errorf("registering informer event handler: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	factory.Start(ctx.Done())

	// Prevent the cache from logging "Waiting for caches to sync"/"Caches are synced".
	ctxLessLogs := klog.NewContext(ctx, klog.Logger{}.V(5))
	if ok := cache.WaitForNamedCacheSyncWithContext(ctxLessLogs, registration.HasSynced); !ok {
		return fmt.Errorf("pod informer did not sync")
	}
	podLister := factory.Core().V1().Pods().Lister()

loop:
	for {
		select {
		case <-notifications:
			done, err := podCondition.Check(podLister)
			if err != nil {
				logger.Warn("error checking pods", "error", err)
				continue
			}
			if done {
				logger.Debug("done waiting", "condition", podCondition)
				return nil
			}
		case <-ctx.Done():
			break loop
		}
	}

	logger.Debug("context expired while waiting", "condition", podCondition)
	// Fetch and print debug information.
	pods, listPodsErr := podLister.List(labels.Everything())
	if listPodsErr != nil {
		logger.Error("could not fetch pods", "error", listPodsErr)
		return err
	}
	for _, pod := range pods {
		logger.Debug("pod status", "name", pod.Name, "status", c.toJSON(pod.Status))
	}
	return fmt.Errorf("context expired while waiting: %w", ctx.Err())
}

// numReady waits until n pods matching ls are ready.
type numReady struct {
	ls labels.Selector
	n  int
}

func (nm *numReady) Check(lister listerscorev1.PodLister) (bool, error) {
	pods, err := lister.List(nm.ls)
	if err != nil {
		return false, err
	}
	n := 0
	for _, pod := range pods {
		if isPodReady(pod) {
			n++
		}
	}
	return n == nm.n, nil
}

func (nm *numReady) String() string {
	return fmt.Sprintf("PodCondition(%d pods matching %s are ready)", nm.n, nm.ls)
}

// numSucceeded waits until n pods matching ls succeeded.
type numSucceeded struct {
	ls labels.Selector
	n  int
}

func (ns *numSucceeded) Check(lister listerscorev1.PodLister) (bool, error) {
	pods, err := lister.List(ns.ls)
	if err != nil {
		return false, err
	}
	n := 0
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodSucceeded {
			n++
		}
	}
	return n >= ns.n, nil
}

func (ns *numSucceeded) String() string {
	return fmt.Sprintf("PodCondition(%d pods matching %s succeeded)", ns.n, ns.ls)
}

// singlePodReady checks that a named pod is ready.
type singlePodReady struct {
	name string
}

func (f *singlePodReady) Check(lister listerscorev1.PodLister) (bool, error) {
	pods, err := lister.List(labels.Everything())
	if err != nil {
		return false, err
	}
	for _, pod := range pods {
		if pod.Name != f.name {
			continue
		}
		return isPodReady(pod), nil
	}
	return false, nil
}

func (f *singlePodReady) String() string {
	return fmt.Sprintf("PodCondition(pod %s is ready)", f.name)
}

type oneRunning struct {
	ls labels.Selector
}

func (or *oneRunning) Check(lister listerscorev1.PodLister) (bool, error) {
	pods, err := lister.List(or.ls)
	if err != nil {
		return false, err
	}
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
	}
	return false, nil
}

func (or *oneRunning) String() string {
	return fmt.Sprintf("PodCondition(one pod matching %s is running)", or.ls)
}
