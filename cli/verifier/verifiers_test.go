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

const podImageRefEmpty = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash"
`

const podTagMissingDigestMalformed = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash@sha256:000"
`

const podTagMissingAlgorithmMissing = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash@0000000000000000000000000000000000000000000000000000000000000000"
`

const podDigestMissing = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash:0.0.1"
`

const podDigestMalformed = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash:0.0.1@sha256:000"
`

const podAlgorithmMissing = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: "bash:0.0.1@0000000000000000000000000000000000000000000000000000000000000000"
`

const podImageRefValid = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
    - name: test
      image: bash:0.0.1@sha256:0000000000000000000000000000000000000000000000000000000000000000
`

func TestVerifyImageRef(t *testing.T) {
	testCases := map[string]struct {
		k8sObjectYAML string
		wantErr       bool
	}{
		"image ref empty": {
			k8sObjectYAML: podImageRefEmpty,
			wantErr:       true,
		},
		"digest malformed, no tag": {
			k8sObjectYAML: podTagMissingDigestMalformed,
			wantErr:       true,
		},
		"digest missing algorithm, no tag": {
			k8sObjectYAML: podTagMissingAlgorithmMissing,
			wantErr:       true,
		},
		"digest missing, with tag": {
			k8sObjectYAML: podDigestMissing,
			wantErr:       true,
		},
		"digest malformed, with tag": {
			k8sObjectYAML: podDigestMalformed,
			wantErr:       true,
		},
		"digest missing algorithm, with tag": {
			k8sObjectYAML: podAlgorithmMissing,
			wantErr:       true,
		},
		"image ref valid": {
			k8sObjectYAML: podImageRefValid,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sObjectYAML))
			require.NoError(err)

			verifier := verifier.ImageRefValid{}

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
