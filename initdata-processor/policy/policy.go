// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"bytes"
	"encoding/json"
	"fmt"

	_ "embed"
)

//go:embed assets/deny-with-message.rego
var denyWithMessagePolicy []byte

// DenyWithMessage returns a Rego policy that answers all Kata agent requests with the given message.
func DenyWithMessage(format string, params ...any) []byte {
	buf := &bytes.Buffer{}
	buf.Write(denyWithMessagePolicy)
	buf.WriteString("\n\nmessage := ")
	fmt.Fprint(buf, regoEscape(fmt.Sprintf(format, params...)))
	return buf.Bytes()
}

// regoEscape returns a double-quoted Rego literal for s.
func regoEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// All strings can be marshalled in Go.
		panic(err)
	}
	return string(b)
}
