/*
The kubeclient package provides a simple wrapper around Kubernetes interactions
commonly used in the e2e tests.
*/
package kubeclient

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// A Kubeclient offers a communication interface to a Kubernetes cluster.
type Kubeclient struct {
	log *slog.Logger

	// client is the underlying Kubernetes client.
	client *kubernetes.Clientset
	// config is the "Kubeconfig" for the client
	config *rest.Config
}

// New creates a new Kubeclient from a given Kubeconfig.
func New(config *rest.Config, log *slog.Logger) (*Kubeclient, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return &Kubeclient{
		log:    log,
		client: client,
		config: config,
	}, nil
}

// NewFromConfigFile creates a new Kubeclient for a given Kubeconfig file.
func NewFromConfigFile(configPath string, log *slog.Logger) (*Kubeclient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("creating config from file: %w", err)
	}

	return New(config, log)
}

// NewForTest creates a Kubeclient with parameters suitable for e2e testing.
func NewForTest(t *testing.T) *Kubeclient {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	configFile := os.Getenv("KUBECONFIG")
	if configFile == "" {
		configFile = clientcmd.RecommendedHomeFile
	}
	c, err := NewFromConfigFile(configFile, log)
	if err != nil {
		t.Fatalf("Could not create Kubeclient: %v", err)
	}
	return c
}

// PodsFromDeployment returns the pods from a deployment in a namespace.
func (c *Kubeclient) PodsFromDeployment(ctx context.Context, namespace, deployment string) ([]v1.Pod, error) {
	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", deployment),
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	return pods.Items, nil
}

// Exec executes a process in a pod and returns the stdout and stderr.
func (c *Kubeclient) Exec(ctx context.Context, namespace, pod string, argv []string) (
	stdout string, stderr string, err error,
) {
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := c.client.CoreV1().RESTClient().
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
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
	if err != nil {
		return "", "", fmt.Errorf("executing command: %w", err)
	}

	return buf.String(), errBuf.String(), nil
}
