// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
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

func run() (retErr error) {
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

	envoyConfig, err := pconfig.ToEnvoyConfig()
	if err != nil {
		return err
	}

	log.Printf("Using envoy configuration:\n%s\n", envoyConfig)

	if err := os.WriteFile(envoyConfigFile, envoyConfig, 0o644); err != nil {
		return err
	}

	if err := IngressIPTableRules(pconfig.ingress); err != nil {
		return fmt.Errorf("failed to set up iptables rules: %w", err)
	}

	// execute the envoy binary
	envoyBin, err := exec.LookPath("envoy")
	if err != nil {
		return err
	}
	_ = envoyBin

	log.Printf("starting ingress proxy")
	cert, err := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if err := ListenAndServe(context.Background(), "0.0.0.0:15006", cfg); err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		if err := ListenAndServe(context.Background(), "0.0.0.0:15007", cfg); err != nil {
			log.Fatal(err)
		}
	}()

	// TODO(burgerdev): proper wait group handling
	time.Sleep(time.Hour)
	log.Printf("ingress proxy done?")

	// TODO(burgerdev): launch envoy

	/*
		log.Println("Starting envoy")
		args := []string{"envoy", "-c", envoyConfigFile}
		args = append(args, os.Args[1:]...)
		return syscall.Exec(envoyBin, args, os.Environ())
	*/
	return nil
}
