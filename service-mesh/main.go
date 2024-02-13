package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

const proxyConfigEnvVar = "EDG_PROXY_CONFIG"

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
	if proxyConfig == "" {
		return fmt.Errorf("no proxy configuration found in environment")
	}

	pconfig, err := ParseProxyConfig(proxyConfig)
	if err != nil {
		return err
	}

	envoyConfig, err := pconfig.ToEnvoyConfig()
	if err != nil {
		return err
	}

	log.Printf("Using envoy configuration:\n%s\n", envoyConfig)

	if err := os.WriteFile("/envoy-config.yaml", envoyConfig, 0o644); err != nil {
		return err
	}

	// execute the envoy binary
	envoyBin, err := exec.LookPath("envoy")
	if err != nil {
		return err
	}

	log.Println("Starting envoy")

	return syscall.Exec(envoyBin, []string{"envoy", "-c", "/envoy-config.yaml"}, os.Environ())
}
