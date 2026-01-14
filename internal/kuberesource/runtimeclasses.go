// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"fmt"
	"strings"

	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// Runtimes returns a set of resources for registering and installing one or multiple runtimes.
func Runtimes(defaultPlatform platforms.Platform, resources []any) ([]any, error) {
	ns := ""
	collectedPlatforms, err := CollectRuntimeClasses(defaultPlatform, resources)
	if err != nil {
		return nil, fmt.Errorf("collecting required runtime classes: %w", err)
	}

	var out []any
	for _, platform := range collectedPlatforms {
		runtimeClass, err := ContrastRuntimeClass(platform)
		if err != nil {
			return nil, fmt.Errorf("creating runtime class for %s: %w", platform.String(), err)
		}
		out = append(out, runtimeClass.RuntimeClassApplyConfiguration)
	}

	nodeInstallers, err := NodeInstallers(ns, collectedPlatforms)
	if err != nil {
		return nil, err
	}

	// Cannot spread the []*DaemonSetApplyConfiguration into []any
	for _, nodeInstaller := range nodeInstallers {
		out = append(out, nodeInstaller)
	}

	return out, nil
}

// CollectRuntimeClasses iterates over all kuberesources and collects the set of used runtime classes.
func CollectRuntimeClasses(defaultPlatform platforms.Platform, resources []any) ([]platforms.Platform, error) {
	collected := []string{defaultPlatform.String()}
	for _, resource := range resources {
		_ = MapPodSpecWithMeta(resource, func(meta *applymetav1.ObjectMetaApplyConfiguration, spec *applycorev1.PodSpecApplyConfiguration,
		) (*applymetav1.ObjectMetaApplyConfiguration, *applycorev1.PodSpecApplyConfiguration) {
			if spec == nil || spec.RuntimeClassName == nil || !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc") || *spec.RuntimeClassName == "contrast-cc" {
				return meta, spec
			}

			collected = append(collected, *spec.RuntimeClassName)
			return meta, spec
		})
	}

	var out []platforms.Platform
	for _, runtimeClass := range removeDuplicateStr(collected) {
		platform, err := platforms.FromRuntimeClassString(runtimeClass)
		if err != nil {
			return nil, err
		}
		out = append(out, platform)
	}

	return out, nil
}

// PatchRuntimeClassName replaces runtime handlers in a set of resources, while allowing for overrides from annotations.
func PatchRuntimeClassName(defaultRuntimeHandler string) func(*applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
	return func(spec *applycorev1.PodSpecApplyConfiguration) *applycorev1.PodSpecApplyConfiguration {
		if spec == nil || spec.RuntimeClassName == nil {
			return spec
		}
		if *spec.RuntimeClassName == "kata-cc-isolation" || *spec.RuntimeClassName == "contrast-cc" {
			spec.RuntimeClassName = &defaultRuntimeHandler
			return spec
		}
		if !strings.HasPrefix(*spec.RuntimeClassName, "contrast-cc-") {
			return spec
		}
		overridePlatform, err := platforms.FromRuntimeClassString(*spec.RuntimeClassName)
		spec.RuntimeClassName = &defaultRuntimeHandler
		if err != nil {
			return spec
		}
		overrideRuntimeHandler, err := manifest.RuntimeHandler(overridePlatform)
		if err != nil {
			return spec
		}
		spec.RuntimeClassName = &overrideRuntimeHandler
		return spec
	}
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
