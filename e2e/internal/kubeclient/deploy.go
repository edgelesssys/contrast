// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// WaitEventCondition is an enum type for the possible wait conditions when using `kubeclient.WaitForEvent`.
type WaitEventCondition int

const (
	_ WaitEventCondition = iota
	// StartingBlocked waits until a specific FailedCreatePodSandBox Event is detected which indicates that the container does not start.
	StartingBlocked
)

// ResourceWaiter is implemented by resources that can be waited for with WaitFor.
type ResourceWaiter interface {
	kind() string
	numDesiredPods(any) (int, error)
	getPods(context.Context, *Kubeclient, string, string) ([]corev1.Pod, error)
}

// Pod implements ResourceWaiter.
type Pod struct{}

func (p Pod) kind() string {
	return "Pod"
}

func (p Pod) numDesiredPods(_ any) (int, error) {
	return 1, nil
}

func (p Pod) getPods(ctx context.Context, client *Kubeclient, namespace, name string) ([]corev1.Pod, error) {
	pod, err := client.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []corev1.Pod{*pod}, nil
}

// Deployment implements ResourceWaiter.
type Deployment struct{}

func (d Deployment) kind() string {
	return "Deployment"
}

func (d Deployment) numDesiredPods(obj any) (int, error) {
	if deploy, ok := obj.(*appsv1.Deployment); ok {
		return int(*deploy.Spec.Replicas), nil
	}
	return 0, fmt.Errorf("watcher received unexpected type %T", obj)
}

func (d Deployment) getPods(ctx context.Context, client *Kubeclient, namespace, name string) ([]corev1.Pod, error) {
	return client.PodsFromDeployment(ctx, namespace, name)
}

// DaemonSet implements ResourceWaiter.
type DaemonSet struct{}

func (d DaemonSet) kind() string {
	return "DaemonSet"
}

func (d DaemonSet) numDesiredPods(obj any) (int, error) {
	if ds, ok := obj.(*appsv1.DaemonSet); ok {
		n := int(ds.Status.DesiredNumberScheduled)
		if n == 0 {
			// DaemonSets start out with empty DesiredNumberScheduled, which then gets filled in by
			// a controller. We don't expect any DaemonSets in our test resources that are
			// intended to be empty, so we artificially require one pod until the status is set
			// correctly.
			n = 1
		}
		return n, nil
	}
	return 0, fmt.Errorf("watcher received unexpected type %T", obj)
}

func (d DaemonSet) getPods(ctx context.Context, client *Kubeclient, namespace, name string) ([]corev1.Pod, error) {
	return client.PodsFromOwner(ctx, namespace, d.kind(), name)
}

// StatefulSet implements ResourceWaiter.
type StatefulSet struct{}

func (s StatefulSet) kind() string {
	return "StatefulSet"
}

func (s StatefulSet) numDesiredPods(obj any) (int, error) {
	if set, ok := obj.(*appsv1.StatefulSet); ok {
		return int(*set.Spec.Replicas), nil
	}
	return 0, fmt.Errorf("watcher received unexpected type %T", obj)
}

func (s StatefulSet) getPods(ctx context.Context, client *Kubeclient, namespace, name string) ([]corev1.Pod, error) {
	return client.PodsFromOwner(ctx, namespace, s.kind(), name)
}

// IsStartingBlocked checks whether the FailedCreatePodSandBox Event occurred which indicates that the SetPolicy request is rejected and the Kata Shim fails to start the Pod sandbox.
func (c *Kubeclient) IsStartingBlocked(name string, namespace string, resource ResourceWaiter, evt watch.Event, startingPoint time.Time) (bool, error) {
	switch evt.Type {
	case watch.Error:
		return false, fmt.Errorf("watcher of %s %s/%s received an error event", resource.kind(), namespace, name)
	case watch.Added:
		fallthrough
	case watch.Modified:
		logger := c.log.With("namespace", namespace)
		event, ok := evt.Object.(*corev1.Event)
		if !ok {
			return false, fmt.Errorf("watcher received unexpected type %T", evt.Object)
		}

		// Expected event: Reason: FailedCreatePodSandBox
		// TODO(jmxnzo): Add patch to the existing error message in Kata Shim, to specifically allow detecting start-up of containers without policy annotation.
		if (event.LastTimestamp.After(startingPoint)) && event.Reason == "FailedCreatePodSandBox" {
			logger.Debug("Pod did not start", "name", name, "reason", event.Reason, "timestamp of failure", event.LastTimestamp.String())
			return true, nil
		}
		return false, nil
	default:
		return false, fmt.Errorf("unexpected watch event while waiting for %s %s/%s: type=%s, object=%#v", resource.kind(), namespace, name, evt.Type, evt.Object)
	}
}

// WaitForService waits until the given service is configured with an external IP and returns it.
func (c *Kubeclient) WaitForService(ctx context.Context, namespace, name string, loadBalancer bool) (string, error) {
	watcher, err := c.Client.CoreV1().Services(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return "", err
	}
	var ip string
	var port int
loop:
	for {
		evt, ok := <-watcher.ResultChan()
		if !ok {
			if ctx.Err() == nil {
				return "", fmt.Errorf("watcher for LoadBalancer %s/%s unexpectedly closed", namespace, name)
			}
			return "", fmt.Errorf("LoadBalancer %s/%s did not get a public IP before %w", namespace, name, ctx.Err())
		}
		switch evt.Type {
		case watch.Added:
			fallthrough
		case watch.Modified:
			svc, ok := evt.Object.(*corev1.Service)
			if !ok {
				return "", fmt.Errorf("watcher received unexpected type %T", evt.Object)
			}
			if loadBalancer {
				for _, ingress := range svc.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						ip = ingress.IP
						// TODO(burgerdev): deal with more than one port, and protocols other than TCP
						port = int(svc.Spec.Ports[0].Port)
						break loop
					}
				}
			} else {
				ip = svc.Spec.ClusterIP
				// TODO(burgerdev): deal with more than one port, and protocols other than TCP
				port = int(svc.Spec.Ports[0].Port)
				break loop
			}
		case watch.Deleted:
			return "", fmt.Errorf("service %s/%s was deleted while waiting for it", namespace, name)
		default:
			c.log.Warn("ignoring unexpected watch event", "type", evt.Type, "object", evt.Object)
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	dialer := &net.Dialer{}
	for {
		select {
		case <-ticker.C:
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
			if err == nil {
				conn.Close()
				return ip, nil
			}
			c.log.Info("probe failed", "namespace", namespace, "name", name, "error", err)
		case <-ctx.Done():
			return "", fmt.Errorf("LoadBalancer %s/%s never responded to probing before %w", namespace, name, ctx.Err())
		}
	}
}

// WaitForEvent watches the EventList as long as the Event corresponding to the WaitCondition occurred after calling the function. It reimplements WaitFor() specifically designed to wait for ocuring events.
func (c *Kubeclient) WaitForEvent(ctx context.Context, condition WaitEventCondition, resource ResourceWaiter, namespace, name string) error {
	// StartingPoint is saved right here to avoid the processing of past events in the checking function! This was introduced, because otherwise calling waitForEvent multiple times
	// resulted in reusing events with the same timestamp.
	startingPoint := time.Now()
	retryCounter := 30
retryLoop:
	for {
		// Watcher which preprocesses the eventList for the defined resource, based on the involvedObject name and the resource kind.
		watcher, err := c.Client.CoreV1().Events(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "involvedObject.name=" + name, TypeMeta: metav1.TypeMeta{Kind: resource.kind()}})
		if err != nil {
			// If the server is down (because K3s was restarted), wait for a
			// second and try again.
			retryCounter--
			if retryCounter != 0 {
				sleep, cancel := context.WithTimeout(ctx, time.Second*1)
				defer cancel()
				<-sleep.Done()
				continue retryLoop
			}
			return err
		}
		for {
			evt, ok := <-watcher.ResultChan()

			if !ok {
				origErr := ctx.Err()
				if origErr == nil {
					retryCounter--
					if retryCounter != 0 {
						continue retryLoop
					}
					return fmt.Errorf("watcher for %s %s/%s unexpectedly closed", resource.kind(), namespace, name)
				}
				logger := c.log.With("namespace", namespace)
				logger.Error("failed to wait for event", "condition", condition, "kind", resource, "name", name, "contextErr", ctx.Err())
				if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return ctx.Err()
				}
				// Fetch and print debug information.
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				pods, err := resource.getPods(ctx, c, namespace, name) //nolint:contextcheck // The parent context expired.
				if err != nil {
					logger.Error("could not fetch pods for resource", "kind", resource.kind(), "name", name, "error", err)
					return origErr
				}
				for _, pod := range pods {
					if !isPodReady(&pod) {
						logger.Debug("pod not ready", "name", pod.Name, "status", c.toJSON(pod.Status))
					}
				}
				return origErr
			}
			switch condition {
			case StartingBlocked:
				blocked, err := c.IsStartingBlocked(name, namespace, resource, evt, startingPoint)
				if err != nil {
					return err
				}
				if blocked {
					return nil
				}
			}
		}
	}
}

func (c *Kubeclient) toJSON(a any) string {
	s, err := json.Marshal(a)
	if err != nil {
		c.log.Error("could not marshal object to JSON", "object", a)
	}
	return string(s)
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (c *Kubeclient) resourceInterfaceFor(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := obj.GroupVersionKind()

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("getting resource for %#v: %w", gvk, err)
	}
	c.log.Info("found mapping", "resource", mapping.Resource)
	ri := c.dyn.Resource(mapping.Resource)
	if mapping.Scope.Name() == "namespace" {
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}
		return ri.Namespace(namespace), nil
	}
	return ri, nil
}

// Apply a set of namespaced manifests to a namespace.
func (c *Kubeclient) Apply(ctx context.Context, objects ...*unstructured.Unstructured) error {
	// Move namespaces to the head of the list so that they are applied first and ready for the other objects.
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].GetKind() == "Namespace" && objects[j].GetKind() != "Namespace"
	})
	for _, obj := range objects {
		ri, err := c.resourceInterfaceFor(obj)
		if err != nil {
			return err
		}
		applied, err := ri.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{Force: true, FieldManager: "e2e-test"})
		if err != nil {
			return fmt.Errorf("could not apply %s %s in namespace %s: %w", obj.GetKind(), obj.GetName(), obj.GetNamespace(), err)
		}
		c.log.Info("object applied", "namespace", applied.GetNamespace(), "kind", applied.GetKind(), "name", applied.GetName())
	}
	return nil
}

// Delete a set of manifests.
func (c *Kubeclient) Delete(ctx context.Context, objects ...*unstructured.Unstructured) error {
	for _, obj := range objects {
		ri, err := c.resourceInterfaceFor(obj)
		if err != nil {
			return err
		}

		if err := ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("could not delete %s %s in namespace %s: %w", obj.GetKind(), obj.GetName(), obj.GetNamespace(), err)
		}
		c.log.Info("object deleted", "namespace", obj.GetNamespace(), "kind", obj.GetKind(), "name", obj.GetName())
	}
	return nil
}

// Restart a resource by deleting all of its dependent pods.
func (c *Kubeclient) Restart(ctx context.Context, resource ResourceWaiter, namespace, name string) error {
	pods, err := resource.getPods(ctx, c, namespace, name)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		err := c.Client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: toPtr(int64(0)),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ScaleDeployment scales a deployment to the given number of replicas.
func (c *Kubeclient) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	_, err := c.Client.AppsV1().Deployments(namespace).UpdateScale(ctx, name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       autoscalingv1.ScaleSpec{Replicas: replicas},
	}, metav1.UpdateOptions{})
	return err
}

// ScaleStatefulSet scales a StatefulSet to the given number of replicas.
func (c *Kubeclient) ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error {
	_, err := c.Client.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       autoscalingv1.ScaleSpec{Replicas: replicas},
	}, metav1.UpdateOptions{})
	return err
}

func toPtr[T any](t T) *T {
	return &t
}
