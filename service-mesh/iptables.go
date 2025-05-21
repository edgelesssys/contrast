// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
)

// EnvoyIngressPort is the port that the envoy proxy listens on for incoming traffic.
const EnvoyIngressPort = 15006

// EnvoyIngressPortNoClientCert is the port that the envoy proxy listens on for
// incoming traffic without requiring a client certificate.
const EnvoyIngressPortNoClientCert = 15007

// IngressIPTableRules sets up the iptables rules for the ingress proxy.
func IngressIPTableRules(ingressEntries []ingressConfigEntry) error {
	iptablesExec, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to create iptables client: %w", err)
	}

	// Reconcile to clean iptables chains.
	// Similar to `ClearChain`, all errors are treated as "chain already exists"
	_ = iptablesExec.NewChain("mangle", "CONTRAST_INBOUND")
	_ = iptablesExec.NewChain("mangle", "CONTRAST_IN_REDIRECT")

	// Route all TCP traffic to the CONTRAST_INBOUND chain.
	if err := iptablesExec.AppendUnique("mangle", "PREROUTING", "-p", "tcp", "-j", "CONTRAST_INBOUND"); err != nil {
		return fmt.Errorf("failed to append CONTRAST_INBOUND chain to PREROUTING chain: %w", err)
	}

	// RETURN all local traffic from the CONTRAST_INBOUND chain back to the PREROUTING chain.
	if err := iptablesExec.AppendUnique("mangle", "CONTRAST_INBOUND", "-p", "tcp", "-i", "lo", "-j", "RETURN"); err != nil {
		return fmt.Errorf("failed to append local traffic exception to CONTRAST_INBOUND chain: %w", err)
	}
	// RETURN all related and established traffic.
	// Since the mangle table executes on every packet and not just before the
	// connection is established, as the nat table does, we need to explicitly
	// return established traffic. Then using tproxy is similar to a REDIRECT
	// rule in the nat table but without the nat overhead.
	// This rule is likely needed to exempt outbound TCP connections.
	// We use "conntrack" instead of "-m socket" as stated in the official
	// documentation because we might not have a kernel with the "xt_socket"
	// module (see: https://github.com/istio/istio/pull/22527).
	// In our own Contrast image the module is available, but we cannot
	// guarantee that it is available in all environments.
	if err := iptablesExec.AppendUnique("mangle", "CONTRAST_INBOUND", "-p", "tcp", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "RETURN"); err != nil {
		return fmt.Errorf("failed to append dport exception to CONTRAST_INBOUND chain: %w", err)
	}
	// Route all other traffic to the CONTRAST_IN_REDIRECT chain.
	if err := iptablesExec.AppendUnique("mangle", "CONTRAST_INBOUND", "-p", "tcp", "-j", "CONTRAST_IN_REDIRECT"); err != nil {
		return fmt.Errorf("failed to append CONTRAST_IN_REDIRECT chain to CONTRAST_INBOUND chain: %w", err)
	}

	for _, entry := range ingressEntries {
		if entry.disableTLS {
			if err := iptablesExec.AppendUnique("mangle", "CONTRAST_IN_REDIRECT", "-p", "tcp", "--dport", fmt.Sprintf("%d", entry.listenPort), "-j", "RETURN"); err != nil {
				return fmt.Errorf("failed to append dport exception to CONTRAST_IN_REDIRECT chain to disable TLS: %w", err)
			}
		} else {
			if err := iptablesExec.AppendUnique("mangle", "CONTRAST_IN_REDIRECT", "-p", "tcp", "--dport", fmt.Sprintf("%d", entry.listenPort), "-j", "TPROXY", "--on-port", fmt.Sprintf("%d", EnvoyIngressPortNoClientCert)); err != nil {
				return fmt.Errorf("failed to append dport exception to CONTRAST_IN_REDIRECT chain to disable client auth: %w", err)
			}
		}
	}

	// Route all remaining traffic (TCP SYN packets that do not have a TLS exemption)
	// to the Envoy proxy port that requires client authentication.
	if err := iptablesExec.AppendUnique("mangle", "CONTRAST_IN_REDIRECT", "-p", "tcp", "-j", "TPROXY", "--on-port", fmt.Sprintf("%d", EnvoyIngressPort)); err != nil {
		return fmt.Errorf("failed to append default TPROXY rule to CONTRAST_IN_REDIRECT chain: %w", err)
	}

	return nil
}
