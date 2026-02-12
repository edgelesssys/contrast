// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"bytes"

	"go.yaml.in/yaml/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// EncodeResources encodes a list of apply configurations into a single YAML document.
func EncodeResources(resources ...any) ([]byte, error) {
	unstructuredResources, err := ResourcesToUnstructured(resources)
	if err != nil {
		return nil, err
	}
	return EncodeUnstructured(unstructuredResources)
}

// ResourcesToUnstructured converts a list of resources into a list of unstructured resources.
func ResourcesToUnstructured(resources []any) ([]*unstructured.Unstructured, error) {
	var unstructuredResources []*unstructured.Unstructured
	for _, r := range resources {
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r)
		if err != nil {
			return nil, err
		}
		unstructuredResources = append(unstructuredResources, &unstructured.Unstructured{Object: u})
	}
	return unstructuredResources, nil
}

// EncodeUnstructured encodes a list of unstructured resources into a single YAML document.
func EncodeUnstructured(resources []*unstructured.Unstructured) ([]byte, error) {
	var w bytes.Buffer
	enc := yaml.NewEncoder(&w)
	enc.SetIndent(2)
	defer enc.Close()
	for _, resource := range resources {
		if err := enc.Encode(resource.Object); err != nil {
			return nil, err
		}
	}

	return w.Bytes(), nil
}
