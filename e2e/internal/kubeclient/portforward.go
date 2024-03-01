package kubeclient

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwardPod starts a port forward to the selected pod.
//
// On success, the function returns a TCP address that clients can connect to and a function to
// cancel the port forwarding.
func (k *Kubeclient) PortForwardPod(ctx context.Context, namespace, podName, remotePort string) (string, func(), error) {
	// This channel sends a stop request to the portforwarding goroutine.
	stopCh := make(chan struct{}, 1)
	// The portforwarding goroutine closes this channel when it's ready.
	readyCh := make(chan struct{})
	// Any error returned by the background port-forwarder is sent to this channel.
	errorCh := make(chan error)

	// Ports are forwarded by upgrading this POST request to a SPDY connection.
	req := k.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(k.config)
	if err != nil {
		return "", nil, fmt.Errorf("creating round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	fw, err := portforward.NewOnAddresses(
		dialer,
		[]string{"localhost"},
		[]string{fmt.Sprintf("0:%s", remotePort)},
		stopCh, readyCh,
		nil, nil,
	)
	if err != nil {
		return "", nil, fmt.Errorf("creating portforwarder: %w", err)
	}

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			errorCh <- err
		}
	}()

	select {
	case <-readyCh:
		ports, err := fw.GetPorts()
		if err != nil {
			close(stopCh)
			return "", nil, fmt.Errorf("getting ports: %w", err)
		}
		cleanUp := func() {
			close(stopCh)
		}
		return fmt.Sprintf("localhost:%d", ports[0].Local), cleanUp, nil

	case <-ctx.Done():
		close(stopCh)
		return "", nil, fmt.Errorf("waiting for port forward to be ready: %w", ctx.Err())
	case err := <-errorCh:
		close(stopCh)
		return "", nil, fmt.Errorf("background port-forwarding routine failed: %w", err)
	}
}
