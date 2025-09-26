// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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
	for i, u := range resources {
		var node yaml.Node
		if err := node.Encode(u.Object); err != nil {
			return nil, err
		}
		setStyles(&node)
		doc, err := yaml.Marshal(&node)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(doc); err != nil {
			return nil, err
		}
		if i != len(resources)-1 {
			if _, err := w.WriteString("---\n"); err != nil {
				return nil, err
			}
		}
	}
	return w.Bytes(), nil
}

func setStyles(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		if strings.Contains(n.Value, "\n") || strings.Contains(n.Value, `\n`) {
			n.Style = yaml.LiteralStyle
		}
	}
	for _, c := range n.Content {
		setStyles(c)
	}
}

// YAMLBytesFromFile reads a k8 YAML file and returns a formatting-preserving encoding.
func YAMLBytesFromFile(yamlPath string) ([]byte, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", yamlPath, err)
	}
	kubeObjs, err := UnmarshalApplyConfigurations(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", yamlPath, err)
	}
	resource, err := EncodeResources(kubeObjs...)
	if err != nil {
		return nil, err
	}
	return resource, nil
}
