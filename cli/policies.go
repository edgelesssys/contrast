package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/katexochen/coordinator-kbs/internal/kubeapi"
	"github.com/katexochen/coordinator-kbs/internal/manifest"
)

func policiesFromKubeResources(yamlPaths []string) (map[string][]byte, error) {
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

	policies := make(map[string][]byte)
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
		policy, err := base64.StdEncoding.DecodeString(annotation)
		if err != nil {
			return nil, fmt.Errorf("failed to decode policy for %s: %w", name, err)
		}
		policies[name] = policy
	}

	return policies, nil
}

func manifestPolicyMapFromPolicies(policies map[string][]byte) (map[manifest.HexString]string, error) {
	policyHashes := make(map[manifest.HexString]string)
	for name, policy := range policies {
		policyHash := sha256.Sum256(policy)
		policyHashStr := manifest.NewHexString(policyHash[:])
		if existingName, ok := policyHashes[policyHashStr]; ok {
			if existingName != name {
				return nil, fmt.Errorf("policy hash collision: %s and %s have the same hash %s",
					existingName, name, policyHashStr)
			}
			continue
		}
		policyHashes[policyHashStr] = name
	}
	return policyHashes, nil
}

func checkPoliciesMatchManifest(policies map[string][]byte, policyHashes map[manifest.HexString]string) error {
	if len(policies) != len(policyHashes) {
		return fmt.Errorf("policy count mismatch: %d policies in deployment, but %d in manifest",
			len(policies), len(policyHashes))
	}
	for name, policy := range policies {
		policyHash := sha256.Sum256(policy)
		policyHashStr := manifest.NewHexString(policyHash[:])
		existingName, ok := policyHashes[policyHashStr]
		if !ok {
			return fmt.Errorf("policy %s not found in manifest", name)
		}
		if existingName != name {
			return fmt.Errorf("policy %s with hash %s exists in manifest, but with different name %s",
				name, policyHashStr, existingName,
			)
		}
	}
	return nil
}
