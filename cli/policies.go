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
		var name, annotation, namespace string
		switch obj := objAny.(type) {
		case kubeapi.Pod:
			name = obj.Name
			namespace = obj.Namespace
			annotation = obj.Annotations[kataPolicyAnnotationKey]
		case kubeapi.Deployment:
			name = obj.Name
			namespace = obj.Namespace
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.ReplicaSet:
			name = obj.Name
			namespace = obj.Namespace
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.StatefulSet:
			name = obj.Name
			namespace = obj.Namespace
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		case kubeapi.DaemonSet:
			name = obj.Name
			namespace = obj.Namespace
			annotation = obj.Spec.Template.Annotations[kataPolicyAnnotationKey]
		}
		if annotation == "" {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("name is required but empty")
		}
		if namespace == "" {
			namespace = "default"
		}
		policy, err := manifest.NewPolicyFromAnnotation([]byte(annotation))
		if err != nil {
			return nil, fmt.Errorf("failed to parse policy %s: %w", name, err)
		}
		deployments[name] = deployment{
			name:      name,
			namespace: namespace,
			policy:    policy,
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
		existingNames, ok := policyHashes[deployment.policy.Hash()]
		if !ok {
			return fmt.Errorf("policy %s not found in manifest", name)
		}

		if !slices.Equal(existingNames, deployment.DNSNames()) {
			return fmt.Errorf("policy %s with hash %s exists in manifest, but with different names %v",
				name, deployment.policy.Hash(), existingNames,
			)
		}
	}
	return nil
}

type deployment struct {
	name      string
	namespace string
	policy    manifest.Policy
}

func (d deployment) DNSNames() []string {
	return []string{
		fmt.Sprintf("%s.%s", d.name, d.namespace),
		fmt.Sprintf("*.%s", d.namespace),
		"*",
	}
}
