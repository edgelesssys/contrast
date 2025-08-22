// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const podWithEmptyVolume = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  volumes:
    - name: asdf
      emptyDir: {}
  containers:
    - name: test
      image: bash
      volumeMounts:
        - name: asdf
          mountPath: /tmp
`

const podWithUnreferencedHostPath = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  volumes:
    - name: unreferenced
      hostPath:
        path: /
  containers:
    - name: test
      image: bash
`

const podWithHostPath = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
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
`

const podWithVolumeMountNoVolume = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: test
      image: bash
      volumeMounts:
        - name: asdf
          mountPath: /tmp
`

const statefulSetWithBlockVolume = `
apiVersion: apps/v1
kind: StatefulSet
spec:
  serviceName: "nginx"
  template:
    spec:
      runtimeClassName: contrast-cc
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
`

const statefulSetWithFSVolume = `
apiVersion: apps/v1
kind: StatefulSet
spec:
  serviceName: "nginx"
  template:
    spec:
      runtimeClassName: contrast-cc
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
`

const nonCCPod = `
apiVersion: v1
kind: Pod
metadata:
  name: test
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
`

const nonCCPodWithHostPath = `
apiVersion: v1
kind: Pod
metadata:
  name: test
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
`

func TestVerifyNoSharedFSMount(t *testing.T) {
	testCases := map[string]struct {
		k8sObjectYAML string
		wantErr       bool
	}{
		"unproblematic yaml": {
			k8sObjectYAML: podWithEmptyVolume,
		},
		"yaml with unreferenced problematic volume": {
			k8sObjectYAML: podWithUnreferencedHostPath,
		},
		"yaml with problematic volume": {
			k8sObjectYAML: podWithHostPath,
			wantErr:       true,
		},
		"yaml with volume mount but no volume": {
			k8sObjectYAML: podWithVolumeMountNoVolume,
			wantErr:       true,
		},
		"stateful set with block volume mode": {
			k8sObjectYAML: statefulSetWithBlockVolume,
		},
		"stateful set with problematic volume mode": {
			k8sObjectYAML: statefulSetWithFSVolume,
			wantErr:       true,
		},
		"non cc pod with good volume": {
			k8sObjectYAML: nonCCPod,
		},
		"non cc pod with bad volume": {
			k8sObjectYAML: nonCCPodWithHostPath,
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

func TestServiceMeshEgress(t *testing.T) {
	testCases := map[string]struct {
		k8sYaml string
		wantErr bool
	}{
		"no annotations": {
			k8sYaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`,
		},
		"good deployment": {
			k8sYaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  annotations:
    contrast.edgeless.systems/servicemesh-egress: "billing#127.137.0.1:8081#billing-svc:8080##cart#127.137.0.2:8081#cart-svc:8080"
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`,
		},
		"bad deployment": {
			k8sYaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  annotations:
    contrast.edgeless.systems/servicemesh-egress: ""
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`,
			wantErr: true,
		},
		"good deployment bad spec": {
			k8sYaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-egress: ""
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`,
			wantErr: true,
		},
		"good deployment good spec": {
			k8sYaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-egress: "asdf"
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sYaml))
			require.NoError(err)

			verifier := verifier.ServiceMeshEgressNotEmpty{}

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
