// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier

import (
	"errors"
	"fmt"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/google/go-containerregistry/pkg/name"
)

// ImageRefValid verifies that all image references contain valid tag and digest.
type ImageRefValid struct{}

// Verify verifies that neither the tag nor the digest of image references are empty.
func (v *ImageRefValid) Verify(toVerify any) error {
	var findings error

	resources, err := kuberesource.ResourcesToUnstructured([]any{toVerify})
	if err != nil {
		return err
	}

	var images []string
	for _, r := range resources {
		findImageFields(r.Object, &images)
	}

	for _, img := range images {
		if _, err := name.NewDigest(img); err != nil {
			findings = errors.Join(findings, fmt.Errorf("the image reference '%s' failed verification. Ensure that it contains a digest and is in the format 'image:tag@sha256:...'", img))
		}
	}

	return findings
}

// findImageFields recursively searches for "image" fields in the unstructured data.
func findImageFields(data map[string]any, images *[]string) {
	for key, value := range data {
		if key == "image" {
			if img, ok := value.(string); ok {
				*images = append(*images, img)
			}
			continue
		}

		switch v := value.(type) {
		case map[string]any:
			findImageFields(v, images)
		case []any:
			for _, item := range v {
				if nestedMap, ok := item.(map[string]any); ok {
					findImageFields(nestedMap, images)
				}
			}
		}

	}
}
