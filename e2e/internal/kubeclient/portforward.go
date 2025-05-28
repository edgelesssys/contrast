// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// WithForwardedPort opens a local port, forwards it to the given pod and invokes the func with the local address.
//
// If the func fails and port-forwarding had an error, too, the func is retried up to two times.
func (k *Kubeclient) WithForwardedPort(ctx context.Context, namespace, podName, remotePort string, f func(addr string) error) error {
	var funcErr error
	for i := range 3 {
		// Apply backoff after first attempt.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i) * time.Second):
		}
		log := k.log.With("attempt", i, "namespace", namespace, "pod", podName, "port", remotePort)

		addr, cancel, errorCh, err := k.portForwardPod(ctx, namespace, podName, remotePort)
		if err != nil {
			log.Error("Could not forward port", "error", err)
			funcErr = err
			continue
		}
		log.Info("forwarded port", "addr", addr)
		funcErr = f(addr)
		cancel()
		if funcErr == nil {
			return nil
		}
		log.Error("port-forwarded func failed", "error", funcErr)
		select {
		case err := <-errorCh:
			log.Error("Encountered port forwarding error", "error", err)
			continue
		default:
			if strings.Contains(funcErr.Error(), "EOF") {
				// Ideally, the condition would use errors.Is(err, io.EOF), but gRPC does not wrap errors.
				log.Info("EOF during port-forwarding triggered retry")
				continue
			}
			log.Info("no port-forwarding error")
			return funcErr
		}
	}
	return funcErr
}

func (k *Kubeclient) portForwardPod(ctx context.Context, namespace, podName, remotePort string) (string, func(), <-chan error, error) {
	// We can only forward to the pod once it's ready.
	if err := k.WaitForPod(ctx, namespace, podName); err != nil {
		return "", nil, nil, fmt.Errorf("waiting for pod %s: %w", podName, err)
	}

	// This channel sends a stop request to the portforwarding goroutine.
	stopCh := make(chan struct{}, 1)
	// The portforwarding goroutine closes this channel when it's ready.
	readyCh := make(chan struct{})
	// Any error returned by the background port-forwarder is sent to this channel.
	errorCh := make(chan error)

	// Ports are forwarded by upgrading this POST request to a SPDY connection.
	req := k.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(k.config)
	if err != nil {
		return "", nil, nil, fmt.Errorf("creating round tripper: %w", err)
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
		return "", nil, nil, fmt.Errorf("creating portforwarder: %w", err)
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
			return "", nil, nil, fmt.Errorf("getting ports: %w", err)
		}
		cleanUp := func() {
			close(stopCh)
		}
		return fmt.Sprintf("localhost:%d", ports[0].Local), cleanUp, errorCh, nil

	case <-ctx.Done():
		close(stopCh)
		return "", nil, nil, fmt.Errorf("waiting for port forward to be ready: %w", ctx.Err())
	case err := <-errorCh:
		close(stopCh)
		return "", nil, nil, fmt.Errorf("background port-forwarding routine failed: %w", err)
	}
}
