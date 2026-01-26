// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/edgelesssys/contrast/internal/platforms"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// Runtimes returns a set of resources for registering and installing one or multiple runtimes.
func Runtimes(defaultPlatform platforms.Platform, resources []any) ([]any, error) {
	ns := ""
	collectedPlatforms, err := CollectRuntimeClasses(resources)
	if err != nil {
		return nil, fmt.Errorf("collecting required runtime classes: %w", err)
	}
	if defaultPlatform != platforms.Unknown {
		collectedPlatforms.Add(defaultPlatform)
	}

	var out []any
	for platform := range collectedPlatforms {
		runtimeClass, err := ContrastRuntimeClass(platform)
		if err != nil {
			return nil, fmt.Errorf("creating runtime class for %s: %w", platform.String(), err)
		}
		out = append(out, runtimeClass.RuntimeClassApplyConfiguration)
	}

	nodeInstallers, err := NodeInstallers(ns, slices.Collect(maps.Keys(collectedPlatforms)))
	if err != nil {
		return nil, err
	}

	// Cannot spread the []*DaemonSetApplyConfiguration into []any
	for _, nodeInstaller := range nodeInstallers {
		out = append(out, nodeInstaller)
	}

	return out, nil
}

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

// CollectRuntimeClasses iterates over all kuberesources and collects the set of used runtime classes.
func CollectRuntimeClasses(resources []any) (PlatformCollection, error) {
	collected := make(PlatformCollection)
	var errs error
	for _, resource := range resources {
		_ = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration,
		) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			if spec == nil || spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc-") {
				return meta, spec
			}
			err := collected.AddFromString(*spec.RuntimeClassName)
			errs = errors.Join(errs, err)
			return meta, spec
		})
	}
	if errs != nil {
		return nil, errs
	}
	return collected, nil
}
