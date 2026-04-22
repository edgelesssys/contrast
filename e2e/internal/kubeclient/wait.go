// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeclient

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
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
	return c.WaitForPodCondition(ctx, namespace, &containerRunning{ls: ls, podName: "coordinator-0", containerName: "coordinator"})
}

// WaitForDaemonSet waits until the DaemonSet is ready.
//
// We consider a DaemonSet to be ready when both:
//   - the count of desired pods is positive
//   - the count of ready pods matches the count of desired pods
//
// Without the first condition, there can be a race between this function and the
// kube-controller-manager, resulting in a wait for 0 pods.
func (c *Kubeclient) WaitForDaemonSet(ctx context.Context, namespace, name string) error {
	var readyDaemonset *appsv1.DaemonSet
	hasStatus := func(ctx context.Context) (done bool, err error) {
		d, err := c.Client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// We don't want to stop waiting in case of transient errors, so we log them and signal
			// that we're not done waiting yet.
			c.log.Warn("Error polling DaemonSet status", "error", err, "namespace", namespace, "name", name)
			return false, nil
		}
		if d.Status.DesiredNumberScheduled < 1 {
			return false, nil
		}
		readyDaemonset = d
		return true, nil
	}
	if err := wait.PollUntilContextCancel(ctx, 2*time.Second, false, hasStatus); err != nil {
		return fmt.Errorf("waiting for Daemonset status: %w", err)
	}
	ls := labels.SelectorFromSet(readyDaemonset.Spec.Selector.MatchLabels)
	return c.WaitForPodCondition(ctx, namespace, &numReady{ls: ls, n: int(readyDaemonset.Status.DesiredNumberScheduled)})
}

// WaitForPod waits until the pod is ready.
func (c *Kubeclient) WaitForPod(ctx context.Context, namespace, name string) error {
	return c.WaitForPodCondition(ctx, namespace, &singlePodReady{name: name})
}

// WaitForContainer waits until a specific container in the named pod is ready.
func (c *Kubeclient) WaitForContainer(ctx context.Context, namespace, podName, containerName string) error {
	return c.WaitForPodCondition(ctx, namespace, &containerReady{podName: podName, containerName: containerName})
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
	Check(PodLister) (bool, error)
}

type PodLister interface {
	List(selector labels.Selector) (ret []*corev1.Pod, err error)
}

type ListerFunc func(selector labels.Selector) (ret []*corev1.Pod, err error)

func (f ListerFunc) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	return f(selector)
}

// WaitForPodCondition waits until the pods in the given namespace satisfy the given condition or the context expires.
func (c *Kubeclient) WaitForPodCondition(ctx context.Context, namespace string, podCondition PodCondition) error {
	logger := c.log.With("namespace", namespace)

	poll := func(ctx context.Context) (done bool, err error) {
		return podCondition.Check(ListerFunc(func(selector labels.Selector) (ret []*corev1.Pod, err error) {
			podList, err := c.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
			if err != nil {
				// TODO(burgerdev): should we just log and return false here?
				return nil, err
			}
			var out []*corev1.Pod
			for i := range podList.Items {
				out = append(out, &podList.Items[i])
			}
			return out, nil
		}))
	}

	err := wait.PollUntilContextCancel(ctx, 2*time.Second /*immediate*/, true, poll)
	if err == nil {
		return nil
	}

	logger.Debug("context expired while waiting", "condition", podCondition)
	// Fetch and print debug information.
	podList, listPodsErr := c.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if listPodsErr != nil {
		logger.Error("could not fetch pods", "error", listPodsErr)
		return err
	}
	for _, pod := range podList.Items {
		logger.Debug("pod status", "name", pod.Name, "status", c.toJSON(pod.Status))
	}
	return fmt.Errorf("context expired while waiting: %w", ctx.Err())
}

// numReady waits until n pods matching ls are ready.
type numReady struct {
	ls labels.Selector
	n  int
}

func (nm *numReady) Check(lister PodLister) (bool, error) {
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

func (ns *numSucceeded) Check(lister PodLister) (bool, error) {
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

func (f *singlePodReady) Check(lister PodLister) (bool, error) {
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

// containerReady checks that a named container in a named pod is ready.
type containerReady struct {
	podName       string
	containerName string
}

func (cr *containerReady) Check(lister PodLister) (bool, error) {
	pods, err := lister.List(labels.Everything())
	if err != nil {
		return false, err
	}
	return checkContainerStatus(pods, cr.podName, cr.containerName, func(cs corev1.ContainerStatus) bool {
		return cs.Ready
	}), nil
}

func (cr *containerReady) String() string {
	return fmt.Sprintf("PodCondition(container %s in pod %s is ready)", cr.containerName, cr.podName)
}

// containerRunning checks that a named container in a named pod is running.
type containerRunning struct {
	ls            labels.Selector
	podName       string
	containerName string
}

func (cr *containerRunning) Check(lister PodLister) (bool, error) {
	pods, err := lister.List(cr.ls)
	if err != nil {
		return false, err
	}
	return checkContainerStatus(pods, cr.podName, cr.containerName, func(cs corev1.ContainerStatus) bool {
		return cs.State.Running != nil
	}), nil
}

func (cr *containerRunning) String() string {
	return fmt.Sprintf("PodCondition(container %s in pod %s is running)", cr.containerName, cr.podName)
}

func checkContainerStatus(pods []*corev1.Pod, podName, containerName string, check func(corev1.ContainerStatus) bool) bool {
	for _, pod := range pods {
		if pod.Name != podName {
			continue
		}
		if pod.DeletionTimestamp != nil {
			return false
		}
		for _, statuses := range [][]corev1.ContainerStatus{
			pod.Status.InitContainerStatuses,
			pod.Status.ContainerStatuses,
			pod.Status.EphemeralContainerStatuses,
		} {
			for _, cs := range statuses {
				if cs.Name == containerName {
					return check(cs)
				}
			}
		}
	}
	return false
}
