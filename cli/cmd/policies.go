// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/initdata"
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

		var annotation string
		var workloadSecretID string
		var role manifest.Role
		switch obj := objAny.(type) {
		case *kubeapi.Pod:
			annotation = obj.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.Deployment:
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.ReplicaSet:
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.StatefulSet:
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.DaemonSet:
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.Job:
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.CronJob:
			name = obj.Name
			annotation = obj.Spec.JobTemplate.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.JobTemplate.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.JobTemplate.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		case *kubeapi.ReplicationController:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[initdata.InitdataAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
		}
		if annotation == "" {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("name is required but empty")
		}
		if workloadSecretID == "" {
			workloadSecretID = strings.Join([]string{orDefault(gvk.Group, "core"), gvk.Version, gvk.Kind, namespace, name}, "/")
		}
		initdata, err := initdata.DecodeKataAnnotation(annotation)
		if err != nil {
			return nil, fmt.Errorf("failed to parse initdata %q: %w", name, err)
		}
		if err := role.Validate(); err != nil {
			return nil, fmt.Errorf("invalid role %s for %s: %w", role, name, err)
		}
		deployments = append(deployments, deployment{
			name:             name,
			initdata:         initdata,
			role:             role,
			workloadSecretID: workloadSecretID,
		})
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
