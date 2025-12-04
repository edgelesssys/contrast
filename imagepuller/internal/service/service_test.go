// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_formatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{
			bytes: 0,
			want:  "0 B",
		},
		{
			bytes: 321,
			want:  "321 B",
		},
		{
			bytes: 4321,
			want:  "4.2 kiB",
		},
		{
			bytes: 54321,
			want:  "53.0 kiB",
		},
		{
			bytes: 654321,
			want:  "639.0 kiB",
		},
		{
			bytes: 7654321,
			want:  "7.3 MiB",
		},
		{
			bytes: 87654321,
			want:  "83.6 MiB",
		},
		{
			bytes: 987654321,
			want:  "941.9 MiB",
		},
		{
			bytes: 9876543210,
			want:  "9.2 GiB",
		},
		{
			bytes: 98765432100,
			want:  "92.0 GiB",
		},
		{
			bytes: 987654321000,
			want:  "919.8 GiB",
		},
		{
			bytes: 9876543210000,
			want:  "9.0 TiB",
		},
		{
			bytes: 98765432100000,
			want:  "89.8 TiB",
		},
		{
			bytes: 10000000000000000000,
			want:  "8.7 EiB",
		},
	}
	for _, tc := range tests {
		name := fmt.Sprintf("%d bytes are %s", tc.bytes, tc.want)
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			got := formatBytes(tc.bytes)
			require.Equal(tc.want, got)
		})
	}
}
