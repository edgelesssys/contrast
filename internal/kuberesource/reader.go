// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"fmt"

	"github.com/edgelesssys/contrast/internal/kubeapi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/applyconfigurations"
)

// UnmarshalApplyConfigurations unmarshals a YAML document into a list of ApplyConfigurations.
func UnmarshalApplyConfigurations(data []byte) ([]any, error) {
	objs, err := kubeapi.UnmarshalUnstructuredK8SResource(data)
	if err != nil {
		return nil, err
	}
	var result []any
	for _, obj := range objs {
		applyConfig := applyconfigurations.ForKind(obj.GetObjectKind().GroupVersionKind())
		if applyConfig == nil {
			return nil, fmt.Errorf("unmarshalling: unsupported resource type %s for %q", obj.GroupVersionKind().String(), obj.GetName())
		}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), applyConfig); err != nil {
			return nil, err
		}
		result = append(result, applyConfig)
	}
	return result, nil
}
