// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package defaultdeny

import (
	"fmt"
	"log/slog"

	"github.com/coreos/go-iptables/iptables"
)

// RemoveDefaultDenyRule removes the default deny rule from the iptables.
func RemoveDefaultDenyRule(log *slog.Logger) error {
	iptablesExec, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to create iptables client: %w", err)
	}

	ruleSpec := []string{"-m", "conntrack", "!", "--ctstate", "ESTABLISHED,RELATED", "-j", "DROP"}

	// Check first if the rule was already deleted. This is the case if the container crashes and is restarted.
	// Not using DeleteIfExists because we want to log if the rule does not exist.
	exists, err := iptablesExec.Exists("filter", "INPUT", ruleSpec...)
	if err != nil {
		return fmt.Errorf("failed to check for default deny rule: %w", err)
	}
	if !exists {
		log.Info("Default deny rule does not exist, nothing to do")
		return nil
	}

	if err := iptablesExec.Delete("filter", "INPUT", ruleSpec...); err != nil {
		return fmt.Errorf("failed to delete default deny rule: %w", err)
	}

	log.Info("Default deny rule removed")

	return nil
}
