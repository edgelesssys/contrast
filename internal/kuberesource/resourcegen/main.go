// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
	addLoadBalancers := flag.Bool("add-load-balancers", false, "Add load balancers to selected services")
	addNamespaceObject := flag.Bool("add-namespace-object", false, "Add namespace object with the given namespace")
	addPortForwarders := flag.Bool("add-port-forwarders", false, "Add port forwarder pods for all services")
	addLogging := flag.Bool("add-logging", false, "Add logging configuration, based on CONTRAST_LOG_LEVEL and CONTRAST_LOG_SUBSYSTEMS environment variables")
	rawPlatform := flag.String("platform", "", "Deployment platform to generate the runtime configuration for")
	addDmesg := flag.Bool("add-dmesg", false, "Add dmesg container")
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
			if *rawPlatform == "" {
				log.Fatalf("--platform must be set to one of %v", platforms.AllStrings())
			}
			var platform platforms.Platform
			platform, err = platforms.FromString(*rawPlatform)
			if err != nil {
				log.Fatalf("Error parsing platform: %v", err)
			}
			subResources, err = kuberesource.Runtime(platform)
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
