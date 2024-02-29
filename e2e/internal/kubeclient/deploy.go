package kubeclient

import (
	"context"
	"fmt"
	"sort"

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
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
			default:
				return fmt.Errorf("unexpected watch event while waiting for pod %s/%s: %#v", namespace, name, evt.Object)
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
				pod, ok := evt.Object.(*appsv1.Deployment)
				if !ok {
					return fmt.Errorf("watcher received unexpected type %T", evt.Object)
				}
				for _, cond := range pod.Status.Conditions {
					if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
			default:
				return fmt.Errorf("unexpected watch event while waiting for deployment %s/%s: %#v", namespace, name, evt.Object)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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
