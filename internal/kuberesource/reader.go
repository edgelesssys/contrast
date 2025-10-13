// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/applyconfigurations"
)

// UnmarshalApplyConfigurations unmarshals a YAML document into a list of ApplyConfigurations.
func UnmarshalApplyConfigurations(data []byte) ([]any, error) {
	objs, err := UnmarshalUnstructuredK8SResource(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling unstructured resources: %w", err)
	}
	var applyConfigs []any
	for _, obj := range objs {
		applyConfig, err := UnstructuredToApplyConfiguration(obj)
		if err != nil {
			return nil, fmt.Errorf("converting unstructured to apply configuration: %w", err)
		}
		applyConfigs = append(applyConfigs, applyConfig)
	}
	return applyConfigs, nil
}

// UnstructuredToApplyConfiguration converts an unstructured resource into a ApplyConfiguration.
func UnstructuredToApplyConfiguration(obj *unstructured.Unstructured) (any, error) {
	applyConfig := applyconfigurations.ForKind(obj.GetObjectKind().GroupVersionKind())
	if applyConfig == nil {
		return nil, fmt.Errorf("unsupported resource type %s for %q", obj.GroupVersionKind().String(), obj.GetName())
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructuredWithValidation(obj.UnstructuredContent(), applyConfig, true); err != nil {
		return nil, fmt.Errorf("converting to %T: %w", applyConfig, err)
	}
	return applyConfig, nil
}

// UnmarshalUnstructuredK8SResource parses the input YAML into unstructured Kubernetes resources.
func UnmarshalUnstructuredK8SResource(data []byte) ([]*unstructured.Unstructured, error) {
	documentsData, err := splitYAML(data)
	if err != nil {
		return nil, fmt.Errorf("splitting YAML into multiple documents: %w", err)
	}

	var objects []*unstructured.Unstructured
	decoder := serializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for idx, documentData := range documentsData {
		obj := &unstructured.Unstructured{}
		_, _, err := decoder.Decode(documentData, nil, obj)
		if err != nil {
			return nil, fmt.Errorf("decoding document %d: %w", idx, err)
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

// YAMLBytesFromFiles reads one or multiple K8s YAML files and returns them in a formatted byte encoding.
func YAMLBytesFromFiles(yamlPaths ...string) ([]byte, error) {
	var kubeObjs []any
	for _, path := range yamlPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		objs, err := UnmarshalApplyConfigurations(data)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling %s: %w", path, err)
		}
		kubeObjs = append(kubeObjs, objs...)
	}
	resource, err := EncodeResources(kubeObjs...)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// splitYAML splits a YAML multidoc into a slice of multiple YAML docs.
func splitYAML(resources []byte) ([][]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(resources))
	var res [][]byte
	for {
		var value any
		err := dec.Decode(&value)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}
