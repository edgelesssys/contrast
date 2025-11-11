// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

/*
Package kubeclient provides a simple wrapper around Kubernetes interactions
commonly used in the e2e tests.
*/
package kubeclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// A Kubeclient offers a communication interface to a Kubernetes cluster.
type Kubeclient struct {
	log *slog.Logger

	// Client is the underlying Kubernetes Client.
	Client *kubernetes.Clientset
	// restMapper allows to look up schema information for dynamic resources
	restMapper meta.RESTMapper
	// config is the "Kubeconfig" for the client
	config *rest.Config
	// dyn is a dynamic API client that can work with unstructured resources
	dyn *dynamic.DynamicClient
}

// New creates a new Kubeclient from a given Kubeconfig.
func New(config *rest.Config, log *slog.Logger) (*Kubeclient, error) {
	client := kubernetes.NewForConfigOrDie(config)
	dyn := dynamic.NewForConfigOrDie(config)

	resources, err := restmapper.GetAPIGroupResources(client.Discovery())
	if err != nil {
		return nil, fmt.Errorf("getting resource groups: %w", err)
	}

	return &Kubeclient{
		log:        log,
		Client:     client,
		config:     config,
		restMapper: restmapper.NewDiscoveryRESTMapper(resources),
		dyn:        dyn,
	}, nil
}

// NewFromConfigFile creates a new Kubeclient for a given Kubeconfig file.
func NewFromConfigFile(configPath string, log *slog.Logger) (*Kubeclient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("creating config from file: %w", err)
	}

	// never use a proxy because otherwise it breaks when we test with an invalid proxy
	config.Proxy = func(*http.Request) (*url.URL, error) { return nil, nil }

	return New(config, log)
}

// NewForTest creates a Kubeclient with parameters suitable for e2e testing.
func NewForTest(t *testing.T) *Kubeclient {
	t.Helper()
	c, err := NewForTestWithoutT()
	if err != nil {
		t.Fatalf("Could not create Kubeclient: %v", err)
	}
	return c
}

// NewForTestWithoutT creates a Kubeclient with parameters suitable for e2e testing.
func NewForTestWithoutT() (*Kubeclient, error) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	configFile := os.Getenv("KUBECONFIG")
	if configFile == "" {
		configFile = clientcmd.RecommendedHomeFile
	}
	c, err := NewFromConfigFile(configFile, log)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// PodsFromDeployment returns the pods from a deployment in a namespace.
//
// A pod is considered to belong to a deployment if it is owned by a ReplicaSet which is in turn
// owned by the Deployment in question. Terminating pods are ignored.
func (c *Kubeclient) PodsFromDeployment(ctx context.Context, namespace, deployment string) ([]corev1.Pod, error) {
	replicasets, err := c.Client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing replicasets: %w", err)
	}
	pods, err := c.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var out []corev1.Pod
	for _, replicaset := range replicasets.Items {
		for _, ref := range replicaset.OwnerReferences {
			if ref.Kind != "Deployment" || ref.Name != deployment {
				continue
			}
			for _, pod := range pods.Items {
				if pod.DeletionTimestamp != nil {
					continue
				}
				for _, ref := range pod.OwnerReferences {
					if ref.Kind == "ReplicaSet" && ref.UID == replicaset.UID {
						out = append(out, pod)
					}
				}
			}
		}
	}

	return out, nil
}

// PodsFromOwner returns the pods owned by an object in the namespace of the given kind.
//
// Terminating pods are ignored.
func (c *Kubeclient) PodsFromOwner(ctx context.Context, namespace, kind, name string) ([]corev1.Pod, error) {
	pods, err := c.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var out []corev1.Pod
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == kind && ref.Name == name {
				out = append(out, pod)
			}
		}
	}

	return out, nil
}

// Exec executes a process in a pod and returns the stdout and stderr.
func (c *Kubeclient) Exec(ctx context.Context, namespace, pod string, argv []string) (
	stdout string, stderr string, err error,
) {
	c.log.Debug("executing command in pod", "namespace", namespace, "pod", pod, "argv", argv)
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := c.Client.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: argv,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(c.config, http.MethodPost, request.URL())
	if err != nil {
		return "", "", fmt.Errorf("creating executor: %w", err)
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
		Tty:    false,
	})

	return buf.String(), errBuf.String(), err
}

// ExecContainer executes a process in the container of a pod and returns the stdout and stderr.
func (c *Kubeclient) ExecContainer(ctx context.Context, namespace, pod, container string, argv []string) (
	stdout string, stderr string, err error,
) {
	c.log.Debug("executing command in container", "namespace", namespace, "pod", pod, "container", container, "argv", argv)
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := c.Client.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   argv,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Container: container,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(c.config, http.MethodPost, request.URL())
	if err != nil {
		return "", "", fmt.Errorf("creating executor: %w", err)
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
		Tty:    false,
	})

	return buf.String(), errBuf.String(), err
}

// ExecDeployment executes a process in one of the deployment's pods.
func (c *Kubeclient) ExecDeployment(ctx context.Context, namespace, deployment string, argv []string) (stdout string, stderr string, err error) {
	if err := c.WaitForDeployment(ctx, namespace, deployment); err != nil {
		return "", "", fmt.Errorf("deployment not ready: %w", err)
	}

	pods, err := c.PodsFromDeployment(ctx, namespace, deployment)
	if err != nil {
		return "", "", fmt.Errorf("could not get pods for deployment %s/%s: %w", namespace, deployment, err)
	}
	if len(pods) == 0 {
		return "", "", fmt.Errorf("no pods found for deployment %s/%s", namespace, deployment)
	}
	c.log.Debug("executing command in deployment pod", "namespace", namespace, "deployment", deployment)
	return c.Exec(ctx, namespace, pods[0].Name, argv)
}

// LogDebugInfo collects pod information from the cluster and writes it to the logger.
func (c *Kubeclient) LogDebugInfo(ctx context.Context) {
	namespaces, err := c.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.log.Error("Could not get namespaces", "error", err)
		return
	}

	for _, namespace := range namespaces.Items {
		c.log.Debug("Collecting debug info for pods", "namespace", namespace.Name)
		pods, err := c.Client.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.log.Error("Could not get pods", "namespace", namespace.Name, "error", err)
			continue
		}
		for _, pod := range pods.Items {
			c.logContainerStatus(pod)
		}
	}
}

func (c *Kubeclient) logContainerStatus(pod corev1.Pod) {
	c.log.Debug("pod status", "name", pod.Name, "namespace", pod.Namespace, "phase", pod.Status.Phase, "reason", pod.Status.Reason, "message", pod.Status.Message)
	for containerType, containers := range map[string][]corev1.ContainerStatus{
		"init":      pod.Status.InitContainerStatuses,
		"main":      pod.Status.ContainerStatuses,
		"ephemeral": pod.Status.EphemeralContainerStatuses,
	} {
		log := c.log.With("pod", pod.Name, "type", containerType)
		for _, container := range containers {
			log.Debug("container status", "name", container.Name, "started", container.Started, "ready", container.Ready, "state", container.State)
		}
	}
}

// GetContainerLogs returns a string holding the kubernetes logs of the specified container.
func (c *Kubeclient) GetContainerLogs(ctx context.Context, resource ResourceWaiter, namespace, name, container string) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		Container: container,
	}
	pods, err := resource.getPods(ctx, c, namespace, name)
	if err != nil {
		return "", fmt.Errorf("failed loading pods:%w", err)
	}
	logStream := c.Client.CoreV1().Pods(namespace).GetLogs(pods[0].Name, &podLogOpts)
	readCloser, err := logStream.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed streaming logging request:%w", err)
	}
	defer readCloser.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, readCloser); err != nil {
		return "", fmt.Errorf("failed copying logs to buffer:%w", err)
	}
	return buf.String(), nil
}
