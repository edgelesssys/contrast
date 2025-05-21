// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/edgelesssys/contrast/internal/defaultdeny"
	"github.com/edgelesssys/contrast/internal/logger"
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

	log, err := logger.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating logger: %v\n", err)
		return err
	}
	defer func() {
		if retErr != nil {
			log.Error(retErr.Error())
		}
	}()

	log.Info("service-mesh started", "version", version)

	egressProxyConfig := os.Getenv(egressProxyConfigEnvVar)
	log.Info("Egress Proxy configuration", "egressProxyConfig", egressProxyConfig)

	ingressProxyConfig := os.Getenv(ingressProxyConfigEnvVar)
	log.Info("Ingress Proxy configuration", "ingressProxyConfig", ingressProxyConfig)

	adminPort := os.Getenv(adminPortEnvVar)
	log.Info("Port for Envoy admin interface", "adminPort", adminPort)

	pconfig, err := ParseProxyConfig(ingressProxyConfig, egressProxyConfig, adminPort)
	if err != nil {
		return err
	}

	envoyConfig, err := pconfig.ToEnvoyConfig()
	if err != nil {
		return err
	}

	log.Info("Using envoy configuration:", "envoyConfig", envoyConfig)

	if err := os.WriteFile(envoyConfigFile, envoyConfig, 0o644); err != nil {
		return err
	}

	if err := IngressIPTableRules(pconfig.ingress); err != nil {
		return fmt.Errorf("failed to set up iptables rules: %w", err)
	}

	// Remove the default deny rule AFTER we set up the configured iptables rules.
	// This way we make sure that all incoming traffic is either blocked by the default deny
	// rule or routed through Envoy as configured by the user.
	if err := defaultdeny.RemoveDefaultDenyRule(log); err != nil {
		return fmt.Errorf("removing default deny rule: %w", err)
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

	log.Info("Starting envoy")
	args := []string{"envoy", "-c", envoyConfigFile}
	args = append(args, os.Args[1:]...)
	return syscall.Exec(envoyBin, args, os.Environ())
}
