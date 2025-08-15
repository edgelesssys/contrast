// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

func TestVerifyNoSharedFSMount(t *testing.T) {
	testCases := map[string]struct {
		k8sObjectYAML string
		wantErr       bool
	}{
		"unproblematic yaml": {
			k8sObjectYAML: `
apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: test
spec:
  volumes:
    - name: asdf
      emptyDir: {}
  containers:
    - name: test
      image: bash
      volumeMounts:
        - name: asdf
          mountPath: /tmp
`,
		},
		"yaml with unreferenced problematic volume": {
			k8sObjectYAML: `
apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: test
spec:
  volumes:
    - name: asdf
      emptyDir: {}
    - name: unreferenced
      hostPath:
        path: /
  containers:
    - name: test
      image: bash
      volumeMounts:
        - name: asdf
          mountPath: /tmp
`,
		},
		"yaml with problematic volume": {
			k8sObjectYAML: `
apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: test
spec:
  volumes:
    - name: asdf
      hostPath:
        path: /
  containers:
    - name: test
      image: bash
      volumeMounts:
        - name: asdf
          mountPath: /tmp
`,
			wantErr: true,
		},
		"stateful set with block volume mode": {
			k8sObjectYAML: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  template:
    metadata:
      labels:
        app: nginx # has to match .spec.selector.matchLabels
    spec:
      containers:
      - name: nginx
        image: registry.k8s.io/nginx-slim:0.24
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      volumeMode: Block
      resources:
        requests:
          storage: 1Gi
`,
		},
		"stateful set with problematic volume mode": {
			k8sObjectYAML: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  template:
    metadata:
      labels:
        app: nginx # has to match .spec.selector.matchLabels
    spec:
      containers:
      - name: nginx
        image: registry.k8s.io/nginx-slim:0.24
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      volumeMode: Filesystem
      resources:
        requests:
          storage: 1Gi
`,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sObjectYAML))
			require.NoError(err)

			verifier := verifier.NoSharedFSMount{}

			for _, toVerify := range toVerifySlice {
				err := verifier.Verify(toVerify)
				if tc.wantErr {
					require.Error(err)
				} else {
					require.NoError(err)
				}
			}
		})
	}
}
