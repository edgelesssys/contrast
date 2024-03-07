package kubeclient

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
