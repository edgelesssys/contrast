// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"golang.org/x/sync/errgroup"
)

const (
	egressProxyConfigEnvVar  = "CONTRAST_EGRESS_PROXY_CONFIG"
	ingressProxyConfigEnvVar = "CONTRAST_INGRESS_PROXY_CONFIG"
	adminPortEnvVar          = "CONTRAST_ADMIN_PORT"
	envoyConfigFile          = "/envoy-config.yml"
)

var version = "0.0.0-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	log.Printf("service-mesh version %s\n", version)

	egressProxyConfig := os.Getenv(egressProxyConfigEnvVar)
	log.Println("Ingress Proxy configuration:", egressProxyConfig)

	ingressProxyConfig := os.Getenv(ingressProxyConfigEnvVar)
	log.Println("Egress Proxy configuration:", ingressProxyConfig)

	adminPort := os.Getenv(adminPortEnvVar)
	log.Println("Port for Envoy admin interface:", adminPort)

	pconfig, err := ParseProxyConfig(ingressProxyConfig, egressProxyConfig, adminPort)
	if err != nil {
		return err
	}

	if err := IngressIPTableRules(pconfig.ingress); err != nil {
		return fmt.Errorf("failed to set up iptables rules: %w", err)
	}

	log.Printf("reading certificates")
	cert, err := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")
	if err != nil {
		return fmt.Errorf("loading key pair: %w", err)
	}

	pool := x509.NewCertPool()
	caCert, err := os.ReadFile("/contrast/tls-config/mesh-ca.pem")
	if err != nil {
		return fmt.Errorf("loading CA certificate: %w", err)
	}
	pool.AppendCertsFromPEM(caCert)

	eg := &errgroup.Group{}

	log.Printf("starting egress proxies")
	for _, egress := range pconfig.egress {
		eg.Go(func() error {
			listener := &Listener{
				Transparent: true,
				Director: func(_ *net.TCPConn) (string, error) {
					return net.JoinHostPort(egress.remoteDomain, strconv.Itoa(int(egress.remotePort))), nil
				},
				UpstreamWrapper: func(conn shutdownConn) shutdownConn {
					return tls.Client(conn, &tls.Config{
						Certificates: []tls.Certificate{cert},
						RootCAs:      pool,
					})
				},
			}
			return listener.ListenAndServe(context.TODO(), net.JoinHostPort(egress.listenAddr.String(), strconv.Itoa(int(egress.listenPort))))
		})
	}

	log.Printf("starting ingress proxies")
	eg.Go(func() error {
		listener := &Listener{
			Transparent: true,
			Director:    originalDst,
			DownstreamWrapper: func(conn shutdownConn) shutdownConn {
				return tls.Server(conn, &tls.Config{
					Certificates: []tls.Certificate{cert},
				})
			},
		}
		return listener.ListenAndServe(context.TODO(), "0.0.0.0:15006")
	})

	eg.Go(func() error {
		listener := &Listener{
			Transparent: true,
			Director:    originalDst,
			DownstreamWrapper: func(conn shutdownConn) shutdownConn {
				return tls.Server(conn, &tls.Config{
					Certificates: []tls.Certificate{cert},
					ClientCAs:    pool,
					ClientAuth:   tls.RequireAndVerifyClientCert,
				})
			},
		}
		return listener.ListenAndServe(context.TODO(), "0.0.0.0:15007")
	})
	return eg.Wait()
}
