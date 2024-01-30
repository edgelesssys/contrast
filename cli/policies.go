package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/edgelesssys/nunki/internal/kubeapi"
	"github.com/edgelesssys/nunki/internal/manifest"
)

func policiesFromKubeResources(yamlPaths []string) (map[string]deployment, error) {
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

	deployments := make(map[string]deployment)
	for _, objAny := range kubeObjs {
		var name, annotation string
		switch obj := objAny.(type) {
		case kubeapi.Pod:
			name = obj.Name
			annotation = obj.Annotations[kataPolicyAnnotationKey]
		case kubeapi.Deployment:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.ReplicaSet:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.StatefulSet:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.DaemonSet:
			name = obj.Name
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
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
		deployments[name] = deployment{
			name:   name,
			policy: policy,
		}
	}

	return deployments, nil
}

func manifestPolicyMapFromPolicies(policies map[string]deployment) (map[manifest.HexString][]string, error) {
	policyHashes := make(map[manifest.HexString][]string)
	for name, depl := range policies {
		if existingNames, ok := policyHashes[depl.policy.Hash()]; ok {
			if slices.Equal(existingNames, depl.DNSNames()) {
				return nil, fmt.Errorf("policy hash collision: %s and %s have the same hash %v",
					existingNames, name, depl.policy.Hash())
			}
			continue
		}
		policyHashes[depl.policy.Hash()] = depl.DNSNames()
	}
	return policyHashes, nil
}

func checkPoliciesMatchManifest(policies map[string]deployment, policyHashes map[manifest.HexString][]string) error {
	if len(policies) != len(policyHashes) {
		return fmt.Errorf("policy count mismatch: %d policies in deployment, but %d in manifest",
			len(policies), len(policyHashes))
	}
	for name, deployment := range policies {
		_, ok := policyHashes[deployment.policy.Hash()]
		if !ok {
			return fmt.Errorf("policy %s not found in manifest", name)
		}
	}
	return nil
}

type deployment struct {
	name   string
	policy manifest.Policy
}

func (d deployment) DNSNames() []string {
	return []string{d.name, "*"}
}
