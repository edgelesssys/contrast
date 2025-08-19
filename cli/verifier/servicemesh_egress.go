// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"

	"github.com/edgelesssys/contrast/internal/kuberesource"
)

// ServiceMeshEgressNotEmpty verifies that the `contrast.edgeless.systems/servicemesh-egress` annotation
// isn't empty if it exists.
type ServiceMeshEgressNotEmpty struct{}

// Verify verifies that the `contrast.edgeless.systems/servicemesh-egress` annotation
// isn't empty if it exists.
func (v *ServiceMeshEgressNotEmpty) Verify(toVerify any) error {
	var findings error

	resources, err := kuberesource.ResourcesToUnstructured([]any{toVerify})
	if err != nil {
		return err
	}
	for _, r := range resources {
		findings = errors.Join(findings, checkUnstructured(r.UnstructuredContent()))
	}

	return findings
}

// checkUnstructured recursively checks if the "contrast.edgeless.systems/servicemesh-egress" annotation has content.
func checkUnstructured(r map[string]any) error {
	var findings error

	for k, v := range r {
		// check if the annotation exists and if it is correct
		if k == "contrast.edgeless.systems/servicemesh-egress" {
			annotation, ok := v.(string)
			if !ok {
				return errors.New("couldn't convert annotation to string")
			}
			if annotation == "" {
				findings = errors.Join(findings, errors.New("empty annotation content in \"contrast.edgeless.systems/servicemesh-egress\", which is likely to be an error"))
			}
			continue
		}
		subMap, ok := v.(map[string]any)
		if ok {
			findings = errors.Join(findings, checkUnstructured(subMap))
		}
		subMapSlice, ok := v.([]map[string]any)
		if ok {
			for _, subMap := range subMapSlice {
				findings = errors.Join(findings, checkUnstructured(subMap))
			}
		}
	}

	return findings
}
