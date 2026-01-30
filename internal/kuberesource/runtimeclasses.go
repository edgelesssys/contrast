// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// PlatformCollection is a helper type to make common operations over the collected, deduplicated set of runtime classes more convenient.
type PlatformCollection map[platforms.Platform]struct{}

// Platforms returns (deduplicated) a slice of all platforms in the PlatformCollection.
func (p *PlatformCollection) Platforms() []platforms.Platform {
	return slices.Collect(maps.Keys(*p))
}

// Names returns a slice of all names in the PlatformCollection.
func (p *PlatformCollection) Names() []string {
	var names []string
	for platform := range *p {
		names = append(names, platform.String())
	}
	return names
}

// Runtimes returns a set of resources for registering and installing one or multiple runtimes.
func (p PlatformCollection) Runtimes() ([]any, error) {
	ns := ""
	var out []any
	for platform := range p {
		runtimeClass, err := ContrastRuntimeClass(platform)
		if err != nil {
			return nil, fmt.Errorf("creating runtime class for %s: %w", platform.String(), err)
		}
		out = append(out, runtimeClass.RuntimeClassApplyConfiguration)
	}

	nodeInstallers, err := NodeInstallers(ns, p.Platforms())
	if err != nil {
		return nil, err
	}

	// Cannot spread the []*DaemonSetApplyConfiguration into []any
	for _, nodeInstaller := range nodeInstallers {
		out = append(out, nodeInstaller)
	}

	return out, nil
}

// Add adds a platform to the collection.
func (p PlatformCollection) Add(platform platforms.Platform) {
	p[platform] = struct{}{}
}

// AddFromString tries to add a platform to the collection from its name.
func (p PlatformCollection) AddFromString(platformName string) error {
	platform, err := platforms.FromRuntimeClassString(platformName)
	if err != nil {
		return err
	}
	p.Add(platform)
	return nil
}

// AddFromCommaSeparated tries to add a platform to the collection from a list of comma-separated names.
func (p PlatformCollection) AddFromCommaSeparated(platformNames string) error {
	for name := range strings.SplitSeq(platformNames, ",") {
		platform, err := platforms.FromString(name)
		if err != nil {
			return err
		}
		p.Add(platform)
	}
	return nil
}

// AddFromResources iterates over all kuberesources and collects the set of used runtime classes.
func (p PlatformCollection) AddFromResources(resources []any) error {
	var errs error
	for _, resource := range resources {
		_ = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration,
		) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			if spec == nil || spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc-") {
				return meta, spec
			}
			err := p.AddFromString(*spec.RuntimeClassName)
			errs = errors.Join(errs, err)
			return meta, spec
		})
	}
	if errs != nil {
		return errs
	}
	return nil
}

// AddFromYamlFiles unmarshals deployment yaml files, then extracts used runtime classes.
func (p PlatformCollection) AddFromYamlFiles(path string) error {
	var deployment []any
	if path != "" {
		yamlFiles, err := CollectYAMLFiles(path)
		if err != nil {
			log.Fatalf("Error collecting deployment files: %v", err)
		}
		yamlBytes, err := YAMLBytesFromFiles(yamlFiles...)
		if err != nil {
			log.Fatalf("Error parsing deployment files: %v", err)
		}
		deployment, err = UnmarshalApplyConfigurations(yamlBytes)
		if err != nil {
			log.Fatalf("Error unmarshaling deployment files: %v", err)
		}
	}
	return p.AddFromResources(deployment)
}
