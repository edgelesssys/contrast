package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

const (
	proxyConfigEnvVar = "EDG_PROXY_CONFIG"
	envoyConfigFile   = "/envoy-config.yml"
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

	proxyConfig := os.Getenv(proxyConfigEnvVar)

	pconfig, err := ParseProxyConfig(proxyConfig)
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

	if err := IngressIPTableRules(); err != nil {
		return fmt.Errorf("failed to set up iptables rules: %w", err)
	}

	// execute the envoy binary
	envoyBin, err := exec.LookPath("envoy")
	if err != nil {
		return err
	}

	log.Println("Starting envoy")

	return syscall.Exec(envoyBin, []string{"envoy", "-c", envoyConfigFile}, os.Environ())
}
