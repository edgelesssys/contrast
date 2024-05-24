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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// WaitForPod watches the given pod and blocks until it meets the condition Ready=True or the
// context expires (is cancelled or times out).
func (c *Kubeclient) WaitForPod(ctx context.Context, namespace, name string) error {
	watcher, err := c.client.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return err
	}
	for {
		select {
		case evt := <-watcher.ResultChan():
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
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WaitForDeployment watches the given deployment and blocks until it meets the condition
// Available=True or the context expires (is cancelled or times out).
func (c *Kubeclient) WaitForDeployment(ctx context.Context, namespace, name string) error {
	watcher, err := c.client.AppsV1().Deployments(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return err
	}
	for {
		select {
		case evt := <-watcher.ResultChan():
			switch evt.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				deploy, ok := evt.Object.(*appsv1.Deployment)
				if !ok {
					return fmt.Errorf("watcher received unexpected type %T", evt.Object)
				}
				pods, err := c.PodsFromDeployment(ctx, namespace, name)
				if err != nil {
					return err
				}
				numPodsReady := 0
				for _, pod := range pods {
					if isPodReady(&pod) {
						numPodsReady++
					}
				}
				if int(*deploy.Spec.Replicas) <= numPodsReady {
					return nil
				}
			case watch.Deleted:
				return fmt.Errorf("deployment %s/%s was deleted while waiting for it", namespace, name)
			default:
				c.log.Warn("ignoring unexpected watch event", "type", evt.Type, "object", evt.Object)
			}
		case <-ctx.Done():
			logger := c.log.With("namespace", namespace)
			logger.Error("deployment did not become ready", "name", name, "contextErr", ctx.Err())
			if ctx.Err() != context.DeadlineExceeded {
				return ctx.Err()
			}
			ctxErr := ctx.Err()
			// Fetch and print debug information.
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			pods, err := c.PodsFromDeployment(ctx, namespace, name) //nolint:contextcheck // The parent context expired.
			if err != nil {
				logger.Error("could not fetch pods for deployment", "name", name, "error", err)
				return ctxErr
			}
			for _, pod := range pods {
				if !isPodReady(&pod) {
					logger.Debug("pod not ready", "name", pod.Name, "status", c.toJSON(pod.Status))
				}
			}
			return ctxErr
		}
	}
}

// WaitForDaemonset watches the given daemonset and blocks until the desired number of pods are
// ready or the context expires (is cancelled or times out).
func (c *Kubeclient) WaitForDaemonset(ctx context.Context, namespace, name string) error {
	watcher, err := c.client.AppsV1().DaemonSets(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return err
	}
	for {
		select {
		case evt := <-watcher.ResultChan():
			switch evt.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				ds, ok := evt.Object.(*appsv1.DaemonSet)
				if !ok {
					return fmt.Errorf("watcher received unexpected type %T", evt.Object)
				}
				if ds.Status.NumberReady >= ds.Status.DesiredNumberScheduled {
					return nil
				}
			default:
				return fmt.Errorf("unexpected watch event while waiting for daemonset %s/%s: %#v", namespace, name, evt.Object)
			}
		case <-ctx.Done():
			logger := c.log.With("namespace", namespace)
			logger.Error("daemonset did not become ready", "name", name, "contextErr", ctx.Err())
			if ctx.Err() != context.DeadlineExceeded {
				return ctx.Err()
			}
			// Fetch and print debug information.
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			pods, err := c.PodsFromOwner(ctx, namespace, "DaemonSet", name) //nolint:contextcheck // The parent context expired.
			if err != nil {
				logger.Error("could not fetch pods for daemonset", "name", name, "error", err)
				return ctx.Err()
			}
			for _, pod := range pods {
				if !isPodReady(&pod) {
					logger.Debug("pod not ready", "name", pod.Name, "status", c.toJSON(pod.Status))
				}
			}
		}
	}
}

// WaitForStatefulSet watches the given StatefulSet and blocks until the desired number of pods are
// ready or the context expires (is cancelled or times out).
func (c *Kubeclient) WaitForStatefulSet(ctx context.Context, namespace, name string) error {
	watcher, err := c.client.AppsV1().StatefulSets(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return err
	}
	for {
		select {
		case evt := <-watcher.ResultChan():
			switch evt.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				set, ok := evt.Object.(*appsv1.StatefulSet)
				if !ok {
					return fmt.Errorf("watcher received unexpected type %T", evt.Object)
				}
				if set.Status.AvailableReplicas >= *set.Spec.Replicas {
					return nil
				}
			default:
				return fmt.Errorf("unexpected watch event while waiting for daemonset %s/%s: %#v", namespace, name, evt.Object)
			}
		case <-ctx.Done():
			logger := c.log.With("namespace", namespace)
			logger.Error("statefulset did not become ready", "name", name, "contextErr", ctx.Err())
			if ctx.Err() != context.DeadlineExceeded {
				return ctx.Err()
			}
			// Fetch and print debug information.
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			pods, err := c.PodsFromOwner(ctx, namespace, "StatefulSet", name) //nolint:contextcheck // The parent context expired.
			if err != nil {
				logger.Error("could not fetch pods for statefulset", "name", name, "error", err)
				return ctx.Err()
			}
			for _, pod := range pods {
				if !isPodReady(&pod) {
					logger.Debug("pod not ready", "name", pod.Name, "status", c.toJSON(pod.Status))
				}
			}
		}
	}
}

// WaitForLoadBalancer waits until the given service is configured with an external IP and returns it.
func (c *Kubeclient) WaitForLoadBalancer(ctx context.Context, namespace, name string) (string, error) {
	watcher, err := c.client.CoreV1().Services(namespace).Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + name})
	if err != nil {
		return "", err
	}
	var ip string
	var port int
loop:
	for {
		select {
		case evt := <-watcher.ResultChan():
			switch evt.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				svc, ok := evt.Object.(*corev1.Service)
				if !ok {
					return "", fmt.Errorf("watcher received unexpected type %T", evt.Object)
				}
				for _, ingress := range svc.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						ip = ingress.IP
						// TODO(burgerdev): deal with more than one port, and protocols other than TCP
						port = int(svc.Spec.Ports[0].Port)
						break loop
					}
				}
			case watch.Deleted:
				return "", fmt.Errorf("service %s/%s was deleted while waiting for it", namespace, name)
			default:
				c.log.Warn("ignoring unexpected watch event", "type", evt.Type, "object", evt.Object)
			}
		case <-ctx.Done():
			return "", fmt.Errorf("LoadBalancer %s/%s did not get a public IP before %w", namespace, name, ctx.Err())
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
	dyn := dynamic.New(c.client.RESTClient())
	gvk := obj.GroupVersionKind()

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("getting resource for %#v: %w", gvk, err)
	}
	c.log.Info("found mapping", "resource", mapping.Resource)
	ri := dyn.Resource(mapping.Resource)
	if namespace := obj.GetNamespace(); namespace != "" {
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

// RestartDeployment restarts a deployment by deleting all of its pods.
func (c *Kubeclient) RestartDeployment(ctx context.Context, namespace, name string) error {
	pods, err := c.PodsFromDeployment(ctx, namespace, name)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		err := c.client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: toPtr(int64(0)),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func toPtr[T any](t T) *T {
	return &t
}
