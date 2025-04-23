// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
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
	fmt.Fprintf(os.Stderr, "Contrast service-mesh %s\n", version)
	fmt.Fprintln(os.Stderr, "Report issues at https://github.com/edgelesssys/contrast/issues")

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

	// Signal readiness for startup probe.
	if err := os.WriteFile("/ready", nil, 0o644); err != nil {
		return err
	}

	// execute the envoy binary
	envoyBin, err := exec.LookPath("envoy")
	if err != nil {
		return err
	}

	log.Println("Starting envoy")
	args := []string{"envoy", "-c", envoyConfigFile}
	args = append(args, os.Args[1:]...)
	return syscall.Exec(envoyBin, args, os.Environ())
}
