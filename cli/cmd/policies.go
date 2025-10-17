// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func policiesFromKubeResources(resources []any) ([]deployment, error) {
	var deployments []deployment
	for _, res := range resources {
		objs, err := kuberesource.ResourcesToUnstructured([]any{res})
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource to unstructured: %w", err)
		}
		if len(objs) != 1 {
			return nil, fmt.Errorf("expected 1 unstructured object, got %d", len(objs))
		}
		objUnstructured := objs[0]

		name := objUnstructured.GetName()
		namespace := orDefault(objUnstructured.GetNamespace(), "default")
		gvk := objUnstructured.GetObjectKind().GroupVersionKind()

		var annotation string
		var workloadSecretID string
		var role manifest.Role
		kuberesource.MapPodSpecWithMeta(res, func(meta *applymetav1.ObjectMetaApplyConfiguration, _ *applycorev1.PodSpecApplyConfiguration) {
			annotation = meta.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(meta.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = meta.Annotations[workloadSecretIDAnnotationKey]
		})
		if annotation == "" {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("name is required but empty")
		}
		if workloadSecretID == "" {
			workloadSecretID = strings.Join([]string{orDefault(gvk.Group, "core"), gvk.Version, gvk.Kind, namespace, name}, "/")
		}
		policy, err := manifest.NewPolicyFromAnnotation([]byte(annotation))
		if err != nil {
			return nil, fmt.Errorf("failed to parse policy %s: %w", name, err)
		}
		if err := role.Validate(); err != nil {
			return nil, fmt.Errorf("invalid role %s for %s: %w", role, name, err)
		}
		deployments = append(deployments, deployment{
			name:             name,
			policy:           policy,
			role:             role,
			workloadSecretID: workloadSecretID,
		})
	}

	return deployments, nil
}

func manifestPolicyMapFromPolicies(policies []deployment) (map[manifest.HexString]manifest.PolicyEntry, error) {
	policyHashes := make(map[manifest.HexString]manifest.PolicyEntry)
	for _, depl := range policies {
		if entry, ok := policyHashes[depl.policy.Hash()]; ok {
			if slices.Equal(entry.SANs, depl.DNSNames()) {
				return nil, fmt.Errorf("policy hash collision: %s and %s have the same hash %v",
					entry.SANs, depl.name, depl.policy.Hash())
			}
			continue
		}
		entry := manifest.PolicyEntry{
			SANs:             depl.DNSNames(),
			WorkloadSecretID: depl.workloadSecretID,
			Role:             depl.role,
		}
		policyHashes[depl.policy.Hash()] = entry
	}
	return policyHashes, nil
}

func checkPoliciesMatchManifest(policies []deployment, policyHashes map[manifest.HexString]manifest.PolicyEntry) error {
	if len(policies) != len(policyHashes) {
		return fmt.Errorf("policy count mismatch: %d policies in deployment, but %d in manifest",
			len(policies), len(policyHashes))
	}
	for _, deployment := range policies {
		_, ok := policyHashes[deployment.policy.Hash()]
		if !ok {
			return fmt.Errorf("policy %s not found in manifest", deployment.name)
		}
	}
	return nil
}

type deployment struct {
	name             string
	policy           manifest.Policy
	role             manifest.Role
	workloadSecretID string
}

func (d deployment) DNSNames() []string {
	return []string{d.name, "*"}
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
