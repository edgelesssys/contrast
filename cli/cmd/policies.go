// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func policiesFromKubeResources(fileMap map[string][]*unstructured.Unstructured) ([]deployment, error) {
	var deployments []deployment
	if err := mapCCWorkloads(fileMap, func(res any, path string, idx int) error {
		name := fileMap[path][idx].GetName()
		namespace := orDefault(fileMap[path][idx].GetNamespace(), "default")
		gvk := fileMap[path][idx].GetObjectKind().GroupVersionKind()

		var annotation string
		var workloadSecretID string
		var role manifest.Role
		kuberesource.MapPodSpecWithMeta(res, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			if meta == nil {
				return meta, spec
			}
			annotation = meta.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(meta.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = meta.Annotations[workloadSecretIDAnnotationKey]
			return meta, spec
		})
		if annotation == "" {
			return fmt.Errorf("missing initdata annotation for %s", name)
		}
		if name == "" {
			return fmt.Errorf("name is required but empty")
		}
		if workloadSecretID == "" {
			workloadSecretID = strings.Join([]string{orDefault(gvk.Group, "core"), gvk.Version, gvk.Kind, namespace, name}, "/")
		}
		initdata, err := initdata.DecodeKataAnnotation(annotation)
		if err != nil {
			return fmt.Errorf("failed to parse initdata %q: %w", name, err)
		}
		if err := role.Validate(); err != nil {
			return fmt.Errorf("invalid role %s for %s: %w", role, name, err)
		}
		deployments = append(deployments, deployment{
			name:             name,
			initdata:         initdata,
			role:             role,
			workloadSecretID: workloadSecretID,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return deployments, nil
}

func manifestPolicyMapFromPolicies(policies []deployment) (map[manifest.HexString]manifest.PolicyEntry, error) {
	policyHashes := make(map[manifest.HexString]manifest.PolicyEntry)
	for _, depl := range policies {
		hash, err := depl.initdata.Digest()
		if err != nil {
			return nil, fmt.Errorf("digesting initdata for %q: %w", depl.name, err)
		}

		if entry, ok := policyHashes[manifest.NewHexString(hash)]; ok {
			if slices.Equal(entry.SANs, depl.DNSNames()) {
				return nil, fmt.Errorf("policy hash collision: %s and %s have the same hash %v",
					entry.SANs, depl.name, manifest.NewHexString(hash))
			}
			continue
		}
		entry := manifest.PolicyEntry{
			SANs:             depl.DNSNames(),
			WorkloadSecretID: depl.workloadSecretID,
			Role:             depl.role,
		}
		policyHashes[manifest.NewHexString(hash)] = entry
	}
	return policyHashes, nil
}

func checkPoliciesMatchManifest(policies []deployment, policyHashes map[manifest.HexString]manifest.PolicyEntry) error {
	if len(policies) != len(policyHashes) {
		return fmt.Errorf("policy count mismatch: %d policies in deployment, but %d in manifest",
			len(policies), len(policyHashes))
	}
	for _, deployment := range policies {
		hash, err := deployment.initdata.Digest()
		if err != nil {
			return fmt.Errorf("digesting initdata: %w", err)
		}
		_, ok := policyHashes[manifest.NewHexString(hash)]
		if !ok {
			return fmt.Errorf("policy %s not found in manifest", deployment.name)
		}
	}
	return nil
}

type deployment struct {
	name             string
	initdata         initdata.Raw
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
