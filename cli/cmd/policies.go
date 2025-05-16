// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/manifest"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type k8sObject interface {
	GetName() string
	GetNamespace() string
	GetObjectKind() schema.ObjectKind
}

func policiesFromKubeResources(yamlPaths []string) ([]deployment, error) {
	var kubeObjs []any
	for _, path := range yamlPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		objs, err := kubeapi.UnmarshalK8SResources(data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}
		kubeObjs = append(kubeObjs, objs...)
	}

	var deployments []deployment
	for _, objAny := range kubeObjs {
		meta, ok := objAny.(k8sObject)
		if !ok {
			continue
		}
		name := meta.GetName()
		namespace := orDefault(meta.GetNamespace(), "default")

		gvk := meta.GetObjectKind().GroupVersionKind()
		workloadSecretID := strings.Join([]string{orDefault(gvk.Group, "core"), gvk.Version, gvk.Kind, namespace, name}, "/")

		var annotation string
		var role manifest.Role
		switch obj := objAny.(type) {
		case *kubeapi.Pod:
			annotation = obj.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.Deployment:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.ReplicaSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.StatefulSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.DaemonSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.Job:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
		case *kubeapi.CronJob:
			name = obj.Name
			annotation = obj.Spec.JobTemplate.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.JobTemplate.Spec.Template.Annotations[contrastRoleAnnotationKey])
		}
		if annotation == "" {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("name is required but empty")
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
