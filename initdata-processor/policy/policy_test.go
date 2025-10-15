// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegoEscape(t *testing.T) {
	for name, tc := range map[string]struct {
		input string
		want  string
	}{
		"empty": {
			input: "",
			want:  `""`,
		},
		"plain ascii": {
			input: "hello world",
			want:  `"hello world"`,
		},
		"double quote": {
			input: `"quoted"`,
			want:  `"\"quoted\""`,
		},
		"backslash": {
			input: `C:\temp\file`,
			want:  `"C:\\temp\\file"`,
		},
		"whitespace": {
			input: "line1\nline2\r\n\tindented",
			want:  `"line1\nline2\r\n\tindented"`,
		},
		"control characters with dedicated escape sequences": {
			input: "\b\f",
			want:  `"\b\f"`,
		},
		"other control characters": {
			input: string([]byte{0x01, 0x02, 0x03}),
			want:  `"\u0001\u0002\u0003"`,
		},
		"injection": {
			input: "foo\"\nCreateContainerRequest { }\nunrelated := \"bar",
			want:  `"foo\"\nCreateContainerRequest { }\nunrelated := \"bar"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := regoEscape(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
