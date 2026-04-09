// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package resolvconf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasNameserver(t *testing.T) {
	for name, tc := range map[string]struct {
		content    string
		wantResult bool
	}{
		"placeholder": {
			content:    "# dummy file, to be bind-mounted by the Kata agent when writing network configuration",
			wantResult: false,
		},
		"only nameserver in comment": {
			content:    "# nameserver 192.0.2.1",
			wantResult: false,
		},
		"nameserver configured": {
			content:    "nameserver 192.0.2.1",
			wantResult: true,
		},
		"nameserver with leading space": {
			content:    "   nameserver 192.0.2.1",
			wantResult: true,
		},
		"nameserver with leading tab": {
			content:    "\tnameserver 192.0.2.1",
			wantResult: true,
		},
		"other configuration": {
			content:    "search foo.bar\n#demo nameserver\nnameserver 192.0.2.1",
			wantResult: true,
		},
		"other configuration without nameserver": {
			content:    "search foo.bar\n#demo nameserver\nmissing",
			wantResult: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantResult, hasNameserver([]byte(tc.content)))
		})
	}
}
