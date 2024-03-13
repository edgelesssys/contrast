package main

import (
	"fmt"
	"os"

	"github.com/coreos/go-iptables/iptables"
)

// EnvoyIngressPort is the port that the envoy proxy listens on for incoming traffic.
const EnvoyIngressPort = 15006

// IngressIPTableRules sets up the iptables rules for the ingress proxy.
func IngressIPTableRules(ingressEntries []ingressConfigEntry) error {
	// Create missing `/run/xtables.lock` file.
	if err := os.Mkdir("/run", 0o755); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create /run directory: %w", err)
		}
	}
	file, err := os.Create("/run/xtables.lock")
	if err != nil {
		return fmt.Errorf("failed to create /run/xtables.lock: %w", err)
	}
	_ = file.Close()

	iptablesExec, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to create iptables client: %w", err)
	}

	// Reconcile to clean iptables chains.
	if err := iptablesExec.ClearChain("mangle", "EDG_INBOUND"); err != nil {
		return fmt.Errorf("failed to clear EDG_INBOUND chain: %w", err)
	}

	if err := iptablesExec.ClearChain("mangle", "EDG_IN_REDIRECT"); err != nil {
		return fmt.Errorf("failed to clear EDG_IN_REDIRECT chain: %w", err)
	}

	// Route all TCP traffic to the EDG_INBOUND chain.
	if err := iptablesExec.AppendUnique("mangle", "PREROUTING", "-p", "tcp", "-j", "EDG_INBOUND"); err != nil {
		return fmt.Errorf("failed to append EDG_INBOUND chain to PREROUTING chain: %w", err)
	}

	// RETURN all local traffic.
	if err := iptablesExec.AppendUnique("mangle", "EDG_INBOUND", "-p", "tcp", "-i", "lo", "-j", "RETURN"); err != nil {
		return fmt.Errorf("failed to append dport exception to EDG_INBOUND chain: %w", err)
	}
	// RETURN all related and established traffic.
	// Since the mangle table executes on every packet and not just before the
	// connection is established, as the nat table does, we need to explicitly
	// return established traffic. Then using tproxy is similar to a REDIRECT
	// rule in the nat table but without the nat overhead.
	// We use "conntrack" instead of "-m socket" as stated in the official
	// documentation because we might not have a kernel with the "xt_socket"
	// module (see: https://github.com/istio/istio/pull/22527)
	if err := iptablesExec.AppendUnique("mangle", "EDG_INBOUND", "-p", "tcp", "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "RETURN"); err != nil {
		return fmt.Errorf("failed to append dport exception to EDG_INBOUND chain: %w", err)
	}
	// Route all other traffic to the EDG_IN_REDIRECT chain.
	if err := iptablesExec.AppendUnique("mangle", "EDG_INBOUND", "-p", "tcp", "-j", "EDG_IN_REDIRECT"); err != nil {
		return fmt.Errorf("failed to append EDG_IN_REDIRECT chain to EDG_INBOUND chain: %w", err)
	}

	for _, entry := range ingressEntries {
		if entry.disableClientCertificate {
			if err := iptablesExec.AppendUnique("mangle", "EDG_IN_REDIRECT", "-p", "tcp", "--dport", fmt.Sprintf("%d", entry.listenPort), "-j", "TPROXY", "--on-port", fmt.Sprintf("%d", 15007)); err != nil {
				return fmt.Errorf("failed to append dport exception to EDG_IN_REDIRECT chain: %w", err)
			}
		}
	}

	// Route all traffic not destined for 127.0.0.1 to the envoy proxy on its
	// port that requires client authentication.
	if err := iptablesExec.AppendUnique("mangle", "EDG_IN_REDIRECT", "-p", "tcp", "-j", "TPROXY", "--on-port", fmt.Sprintf("%d", EnvoyIngressPort)); err != nil {
		return fmt.Errorf("failed to append EDG_IN_REDIRECT chain to TPROXY chain: %w", err)
	}

	return nil
}
