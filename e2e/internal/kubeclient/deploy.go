// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kubeclient

import (
	"context"
	"encoding/json"
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
	"k8s.io/client-go/kubernetes"
)

type WaitCondition int

const (
	_ WaitCondition = iota
	Ready
	Added
	Modified
	Deleted
	Bookmark
	Running
)

// ResourceWaiter is implemented by resources that can be waited for with WaitFor.
type ResourceWaiter interface {
	kind() string
	watcher(context.Context, *kubernetes.Clientset, string, string) (watch.Interface, error)
	numDesiredPods(any) (int, error)
	getPods(context.Context, *Kubeclient, string, string) ([]corev1.Pod, error)
}

// Pod implements ResourceWaiter.
type Pod struct{}

func (p Pod) kind() string {
	return "Pod"
}

func (p Pod) watcher(ctx context.Context, client *kubernetes.Clientset, namespace, name string) (watch.Interface, error) {
	return client.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
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

func (d Deployment) watcher(ctx context.Context, client *kubernetes.Clientset, namespace, name string) (watch.Interface, error) {
	return client.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
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

func (d DaemonSet) watcher(ctx context.Context, client *kubernetes.Clientset, namespace, name string) (watch.Interface, error) {
	return client.AppsV1().DaemonSets(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
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

func (s StatefulSet) watcher(ctx context.Context, client *kubernetes.Clientset, namespace, name string) (watch.Interface, error) {
	return client.AppsV1().StatefulSets(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
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

// WaitForPod watches the given pod and blocks until it meets the condition Ready=True or the
// context expires (is cancelled or times out).
func (c *Kubeclient) WaitForPod(ctx context.Context, namespace, name string) error {
	watcher, err := c.Client.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return err
	}
	for {
		evt, ok := <-watcher.ResultChan()
		if !ok {
			if ctx.Err() == nil {
				return fmt.Errorf("watcher for Pod %s/%s unexpectedly closed", namespace, name)
			}
			return ctx.Err()
		}
		switch evt.Type {
		case watch.Added:
			fallthrough
		case watch.Modified:
			pod, ok := evt.Object.(*corev1.Pod)
			if !ok {
				return fmt.Errorf("watcher received unexpected type %T", evt.Object)
			}
			if isPodReady(pod) {
				return nil
			}
		case watch.Deleted:
			return fmt.Errorf("pod %s/%s was deleted while waiting for it", namespace, name)
		default:
			c.log.Warn("ignoring unexpected watch event", "type", evt.Type, "object", evt.Object)
		}
	}
}

// WaitFor watches the given resource kind and blocks until the desired number of pods are
// ready or the context expires (is cancelled or times out).
func (c *Kubeclient) WaitFor(ctx context.Context, resource ResourceWaiter, namespace, name string) error {
	logger := c.log.With("namespace", namespace)
	logger.Info(fmt.Sprintf("Waiting for %s %s/%s to become ready", resource.kind(), namespace, name))

	// When the node-installer restarts K3s, the watcher fails. The watcher has
	// a retry loop internally, but it only retries starting the request, once
	// it has established a request and that request dies spuriously, the
	// watcher doesn't reconnect. To fix this we add another retry loop.
	retryCounter := 30

retryLoop:
	for {
		watcher, err := resource.watcher(ctx, c.Client, namespace, name)
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
				logger.Error("resource did not become ready", "kind", resource, "name", name, "contextErr", ctx.Err())
				if ctx.Err() != context.DeadlineExceeded {
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
			switch evt.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				pods, err := resource.getPods(ctx, c, namespace, name)
				if err != nil {
					return err
				}
				numPodsReady := 0
				for _, pod := range pods {
					if isPodReady(&pod) {
						numPodsReady++
					}
				}
				desiredPods, err := resource.numDesiredPods(evt.Object)
				if err != nil {
					return err
				}
				if desiredPods <= numPodsReady {
					// Wait for 5 more seconds just to be *really* sure that
					// the pods are actually up.
					sleep, cancel := context.WithTimeout(ctx, time.Second*5)
					defer cancel()
					<-sleep.Done()
					return nil
				}
			case watch.Deleted:
				return fmt.Errorf("%s %s/%s was deleted while waiting for it", resource.kind(), namespace, name)
			default:
				return fmt.Errorf("unexpected watch event while waiting for %s %s/%s: type=%s, object=%#v", resource.kind(), namespace, name, evt.Type, evt.Object)
			}
		}
	}
}

// WaitFor watches the given resource kind and blocks until the desired number of pods are
// ready or the context expires (is cancelled or times out).
func (c *Kubeclient) WaitFor(ctx context.Context, condition WaitCondition, resource ResourceWaiter, namespace, name string) error {
	switch condition {
	case Ready:
		return c.waitForReady(ctx, name, namespace, resource)
	case Added:
		// TODO
	case Modified:
		// TODO
	case Deleted:
		// TODO
	case Running:
		// TODO
	}
	return fmt.Errorf("Provided wait condition is not supported")
}

// WaitForLoadBalancer waits until the given service is configured with an external IP and returns it.
func (c *Kubeclient) WaitForLoadBalancer(ctx context.Context, namespace, name string) (string, error) {
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
	dyn := dynamic.New(c.Client.RESTClient())
	gvk := obj.GroupVersionKind()

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("getting resource for %#v: %w", gvk, err)
	}
	c.log.Info("found mapping", "resource", mapping.Resource)
	ri := dyn.Resource(mapping.Resource)
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

func toPtr[T any](t T) *T {
	return &t
}
