// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/platforms"
)

func main() {
	imageReplacementsPath := flag.String("image-replacements", "", "Path to the image replacements file")
	namespace := flag.String("namespace", "", "Namespace for namespaced resources")
	rawPlatform := flag.String("platform", "", "Deployment platform to generate the runtime configuration for")
	addLoadBalancers := flag.Bool("add-load-balancers", false, "Add load balancers to selected services")
	addNamespaceObject := flag.Bool("add-namespace-object", false, "Add namespace object with the given namespace")
	addPortForwarders := flag.Bool("add-port-forwarders", false, "Add port forwarder pods for all services")
	addLogging := flag.Bool("add-logging", false, "Add logging configuration, based on CONTRAST_LOG_LEVEL and CONTRAST_LOG_SUBSYSTEMS environment variables")
	addDmesg := flag.Bool("add-dmesg", false, "Add dmesg container")
	nodeInstallerTargetConfType := flag.String("node-installer-target-conf-type", "", "Type of node installer target configuration to generate (k3s,...)")
	deploymentPath := flag.String("deployment", "", "Path to the deployment file or a folder containing the deployment file(s)")
	gpuClass := flag.String("gpu-class", "nvidia.com/pgpu", "full vendor/class of the GPU to attach")
	gpuCount := flag.Int("gpu-count", 1, "number of GPUs to attach")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <set>...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	var resources []any
	for _, set := range flag.Args() {
		var subResources []any
		var err error
		switch set {
		case "coordinator":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.CoordinatorBundle(), "contrast-cc")
		case "runtime":
			var defaultPlatform platforms.Platform
			defaultPlatform, err = platforms.FromStringOrEmpty(*rawPlatform)
			if err != nil {
				log.Fatalf("Error parsing platform: %v", err)
			}

			var deployment []any
			if *deploymentPath != "" {
				yamlFiles, err := kuberesource.CollectYAMLFiles(*deploymentPath)
				if err != nil {
					log.Fatalf("Error collecting deployment files: %v", err)
				}
				yamlBytes, err := kuberesource.YAMLBytesFromFiles(yamlFiles...)
				if err != nil {
					log.Fatalf("Error parsing deployment files: %v", err)
				}
				deployment, err = kuberesource.UnmarshalApplyConfigurations(yamlBytes)
				if err != nil {
					log.Fatalf("Error unmarshalling deployment files: %v", err)
				}
			}

			subResources, err = kuberesource.Runtimes(defaultPlatform, deployment)
		case "node-installer-target-conf":
			if *nodeInstallerTargetConfType == "" {
				log.Fatalf("--node-installer-target-conf-type must be set")
			}
			subResources = make([]any, 1)
			subResources[0], err = kuberesource.NodeInstallerTargetConfig(*nodeInstallerTargetConfType)
		case "openssl":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.OpenSSL(), "contrast-cc")
		case "emojivoto":
			subResources = kuberesource.Emojivoto(kuberesource.ServiceMeshDisabled)
			subResources = kuberesource.PatchRuntimeHandlers(subResources, "contrast-cc")
		case "emojivoto-sm-ingress":
			subResources = kuberesource.Emojivoto(kuberesource.ServiceMeshIngressEgress)
			subResources = kuberesource.PatchRuntimeHandlers(subResources, "contrast-cc")
		case "volume-stateful-set":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.VolumeStatefulSet(), "contrast-cc")
		case "mysql":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.MySQL(), "contrast-cc")
		case "vault":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.Vault(*namespace), "contrast-cc")
		case "gpu":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.GPU("gpu-tester", *gpuClass, int64(*gpuCount)), "contrast-cc")
		default:
			log.Fatalf("Error: unknown set: %s\n", set)
		}
		if err != nil {
			log.Fatalf("Error generating %q: %v", set, err)
		}
		resources = append(resources, subResources...)
	}

	if *addPortForwarders {
		resources = kuberesource.AddPortForwarders(resources)
	}

	if *addLoadBalancers {
		resources = kuberesource.AddLoadBalancers(resources)
	}

	if *addDmesg {
		resources = kuberesource.AddDmesg(resources)
	}

	if *addLogging {
		logLevel := os.Getenv("CONTRAST_LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
		logSubSystems := os.Getenv("CONTRAST_LOG_SUBSYSTEMS")
		if logSubSystems == "" {
			logSubSystems = "*"
		}
		resources = kuberesource.AddLogging(resources, logLevel, logSubSystems)
	}

	var replacements map[string]string
	if *imageReplacementsPath != "" {
		f, err := os.Open(*imageReplacementsPath)
		if err != nil {
			log.Fatalf("could not open image definition file %q: %v", *imageReplacementsPath, err)
		}
		defer f.Close()

		replacements, err = kuberesource.ImageReplacementsFromFile(f)
		if err != nil {
			log.Fatalf("could not parse image definition file %q: %v", *imageReplacementsPath, err)
		}
	}

	kuberesource.PatchImages(resources, replacements)
	if *namespace != "" {
		kuberesource.PatchNamespaces(resources, *namespace)
		if *addNamespaceObject && *namespace != "default" {
			resources = append([]any{kuberesource.Namespace(*namespace)}, resources...)
		}
	}

	b, err := kuberesource.EncodeResources(resources...)
	if err != nil {
		log.Fatalf("Error encoding resources: %v", err)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		log.Fatalf("Error writing to stdout: %v", err)
	}
}
