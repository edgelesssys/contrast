// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package peerdiscovery

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPeers(t *testing.T) {
	host := "coordinator-0"
	namespace := "test"

	testCases := map[string]struct {
		pods     []runtime.Object
		expected []string
	}{
		"no pods": {
			pods:     nil,
			expected: nil,
		},
		"no peers": {
			pods: []runtime.Object{
				newPod(host, namespace, "coordinator", "coordinator", "1.2.3.4", true),
			},
			expected: nil,
		},
		"multiple peers": {
			pods: []runtime.Object{
				newPod(host, namespace, "coordinator", "coordinator", "1.2.3.4", true),
				newPod("coordinator-1", namespace, "coordinator", "coordinator", "5.6.7.8", true),
				newPod("coordinator-2", namespace, "coordinator", "coordinator", "9.10.11.12", true),
			},
			expected: []string{"5.6.7.8", "9.10.11.12"},
		},
		"peer not ready": {
			pods: []runtime.Object{
				newPod(host, namespace, "coordinator", "coordinator", "1.2.3.4", true),
				newPod("coordinator-1", namespace, "coordinator", "coordinator", "5.6.7.8", false),
			},
			expected: nil,
		},
		"peer has no coordinator role": {
			pods: []runtime.Object{
				newPod(host, namespace, "coordinator", "coordinator", "1.2.3.4", true),
				newPod("coordinator-1", namespace, "coordinator", "worker", "5.6.7.8", true),
			},
			expected: nil,
		},
		"peer has no coordinator label": {
			pods: []runtime.Object{
				newPod(host, namespace, "coordinator", "coordinator", "1.2.3.4", true),
				newPod("coordinator-1", namespace, "worker", "coordinator", "5.6.7.8", true),
			},
			expected: nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			t.Setenv("HOSTNAME", host)

			client := fake.NewSimpleClientset(tc.pods...)
			peers, err := New(client, namespace).GetPeers(context.Background())
			require.NoError(err)
			slices.Sort(tc.expected)
			slices.Sort(peers)
			require.Equal(tc.expected, peers)
		})
	}
}

func newPod(name, namespace, labelName, role, ip string, isReady bool) runtime.Object {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      map[string]string{"app.kubernetes.io/name": labelName},
			Annotations: map[string]string{"contrast.edgeless.systems/pod-role": role},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}},
			PodIP:      ip,
		},
	}
	if isReady {
		pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	}
	return pod
}
