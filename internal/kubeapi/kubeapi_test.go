// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kubeapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalK8SResource(t *testing.T) {
	testCases := map[string]struct {
		resources string
		wantErr   bool
		wantTypes []any
	}{
		"namespace ignored": {
			wantTypes: []any{},
			resources: `
apiVersion: v1
kind: Namespace
metadata:
    name: test
`,
		},
		"pod": {
			wantTypes: []any{&Pod{}},
			resources: `
apiVersion: v1
kind: Pod
metadata:
    name: test
`,
		},
		"deployment, ignored service, daemonset": {
			wantTypes: []any{&Deployment{}, &DaemonSet{}},
			resources: `
apiVersion: apps/v1
kind: Deployment
metadata:
    name: test
---
apiVersion: v1
kind: Service
metadata:
    name: test
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
    name: test
`,
		},
		"statefulset, replicaset": {
			wantTypes: []any{&StatefulSet{}, &ReplicaSet{}},
			resources: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
    name: test
---
apiVersion: apps/v1
kind: ReplicaSet
metadata:
    name: test
`,
		},
		"job": {
			wantTypes: []any{&Job{}},
			resources: `
apiVersion: batch/v1
kind: Job
metadata:
    name: test
`,
		},
		"cronjob": {
			wantTypes: []any{&CronJob{}},
			resources: `
apiVersion: batch/v1
kind: CronJob
metadata:
    name: test
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			got, err := UnmarshalK8SResources([]byte(tc.resources))

			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			require.Len(got, len(tc.wantTypes))
			for i, g := range got {
				assert.IsType(tc.wantTypes[i], g)
			}
		})
	}
}

func TestSplitYAML(t *testing.T) {
	testCases := map[string]struct {
		resources string
		wantSplit []string
		wantErr   bool
	}{
		"empty": {
			resources: "",
			wantSplit: []string{},
		},
		"single resource": {
			resources: `apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			},
		},
		"single resource with doc separator": {
			resources: `
---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
			},
		},
		"2 documents": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
---
apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
			},
		},
		"3 documents": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
    name: test1
---
apiVersion: v1
kind: Namespace
metadata:
    name: test2
---
apiVersion: v1
kind: Namespace
metadata:
    name: test3
`,
			wantSplit: []string{
				`apiVersion: v1
kind: Namespace
metadata:
    name: test1
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test2
`,
				`apiVersion: v1
kind: Namespace
metadata:
    name: test3
`,
			},
		},
		"2 document invalid": {
			resources: `---
apiVersion: v1
kind: Namespace
metadata:
	name: test1
---
apiVersion
`,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotSplit, err := splitYAML([]byte(tc.resources))

			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tc.wantSplit, bytesToStrings(gotSplit))
		})
	}
}

func bytesToStrings(b [][]byte) []string {
	s := make([]string, len(b))
	for i, bb := range b {
		s[i] = string(bb)
	}
	return s
}
