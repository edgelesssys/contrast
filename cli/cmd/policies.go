// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"log/slog"
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

// volumeInfo holds the tally of the volume, as well as the type and the names of resources using the volume.
type volumeInfo struct {
	mountCount    int
	resourceNames []string
}

func policiesFromKubeResources(yamlPaths []string, logger *slog.Logger) ([]deployment, error) {
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
	volumeInfoMap := make(map[string]*volumeInfo)
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
			annotation = obj.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&kubeapi.PodTemplateSpec{
				Spec: obj.Spec,
			}, name, volumeInfoMap)
		case *kubeapi.Deployment:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.Template, name, volumeInfoMap)
		case *kubeapi.ReplicaSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.Template, name, volumeInfoMap)
		case *kubeapi.StatefulSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.Template, name, volumeInfoMap)
		case *kubeapi.DaemonSet:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.Template, name, volumeInfoMap)
		case *kubeapi.Job:
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.Template, name, volumeInfoMap)
		case *kubeapi.CronJob:
			name = obj.Name
			annotation = obj.Spec.JobTemplate.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.JobTemplate.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.JobTemplate.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(&obj.Spec.JobTemplate.Spec.Template, name, volumeInfoMap)
		case *kubeapi.ReplicationController:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
			role = manifest.Role(obj.Spec.Template.Annotations[contrastRoleAnnotationKey])
			workloadSecretID = obj.Spec.Template.Annotations[workloadSecretIDAnnotationKey]
			accumulateVolumeMounts(obj.Spec.Template, name, volumeInfoMap)
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
	reportVolumeSharing(logger, volumeInfoMap)

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

// accumulateVolumeMounts updates the shared volumeInfoMap for a given PodTemplateSpec.
// It maps each declared volume to a key ("<resourceName>:emptyDir:<volName>" or "<type>:<volName>"),
// ensures a volumeInfo entry exists, and tallies mounts from all containers in the given PodTemplateSpec.
func accumulateVolumeMounts(
	podSpec *kubeapi.PodTemplateSpec,
	resourceName string,
	volumes map[string]*volumeInfo,
) {
	volumeKey := make(map[string]string, len(podSpec.Spec.Volumes))

	// register every declared volume and pick its key
	for _, vol := range podSpec.Spec.Volumes {
		var key string
		switch {
		case vol.EmptyDir != nil:
			// emptyDirs get a per‐pod prefix so they never collide
			key = fmt.Sprintf("%s:emptyDir:%s", resourceName, vol.Name)

		case vol.PersistentVolumeClaim != nil:
			key = fmt.Sprintf("pvc:%s", vol.Name)

		case vol.ConfigMap != nil:
			key = fmt.Sprintf("configMap:%s", vol.Name)
		default:
			key = fmt.Sprintf("other:%s", vol.Name)
		}

		volumeKey[vol.Name] = key

		if _, seen := volumes[key]; !seen {
			volumes[key] = &volumeInfo{}
		}
	}

	// helper: look up the key and bump count + record resourceName
	record := func(volName string) {
		key, ok := volumeKey[volName]
		if !ok {
			// ignore any mounts of undeclared volumes
			return
		}
		vi := volumes[key]
		vi.mountCount++

		// append this resource once
		for _, r := range vi.resourceNames {
			if r == resourceName {
				return
			}
		}
		vi.resourceNames = append(vi.resourceNames, resourceName)
	}

	for _, ctr := range podSpec.Spec.Containers {
		for _, vm := range ctr.VolumeMounts {
			record(vm.Name)
		}
	}
	for _, initCtr := range podSpec.Spec.InitContainers {
		for _, vm := range initCtr.VolumeMounts {
			record(vm.Name)
		}
	}
}

// reportVolumeSharing logs one warning per shared volume inside and across pods,
// extracting both the volume type and real volume name from the key.
//
// Keys are either:
//
//	"<resourceName>:emptyDir:<volName>"
//
// or:
//
//	"<type>:<volName>"
func reportVolumeSharing(
	logger *slog.Logger,
	volumes map[string]*volumeInfo,
) {
	for key, vi := range volumes {
		if vi.mountCount <= 1 {
			continue
		}
		parts := strings.Split(key, ":")
		actualVol := parts[len(parts)-1]
		volType := parts[len(parts)-2]

		resList := strings.Join(vi.resourceNames, ", ")
		if len(vi.resourceNames) == 1 {
			logger.Warn(fmt.Sprintf(
				"resource %s: %s volume `%s` is shared by %d containers on one pod",
				vi.resourceNames[0], volType, actualVol, vi.mountCount,
			))
		} else {
			base := fmt.Sprintf(
				"%s volume `%s` is shared by %d containers across pods in resources [%s]", volType,
				actualVol, vi.mountCount, resList,
			)
			logger.Warn(base)
		}
	}
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
