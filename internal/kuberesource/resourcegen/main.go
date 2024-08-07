// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
)

func main() {
	imageReplacementsPath := flag.String("image-replacements", "", "Path to the image replacements file")
	namespace := flag.String("namespace", "", "Namespace for namespaced resources")
	addLoadBalancers := flag.Bool("add-load-balancers", false, "Add load balancers to selected services")
	addNamespaceObject := flag.Bool("add-namespace-object", false, "Add namespace object with the given namespace")
	addPortForwarders := flag.Bool("add-port-forwarders", false, "Add port forwarder pods for all services")
	rawPlatform := flag.String("platform", "", "Deployment platform to generate the runtime configuration for")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <set>...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *rawPlatform == "" {
		log.Fatalf("--platform must be set to one of %v", platforms.AllStrings())
	}

	platform, err := platforms.FromString(*rawPlatform)
	if err != nil {
		log.Fatalf("Error parsing platform: %v", err)
	}

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	if err != nil {
		log.Fatalf("Error getting default runtime handler: %v", err)
	}

	var resources []any
	for _, set := range flag.Args() {
		var subResources []any
		var err error
		switch set {
		case "coordinator":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.CoordinatorBundle(), runtimeHandler)
		case "runtime":
			subResources, err = kuberesource.Runtime(platform)
		case "openssl":
			subResources = kuberesource.PatchRuntimeHandlers(kuberesource.OpenSSL(), runtimeHandler)
		case "emojivoto":
			subResources = kuberesource.Emojivoto(kuberesource.ServiceMeshDisabled)
			subResources = kuberesource.PatchRuntimeHandlers(subResources, runtimeHandler)
		case "emojivoto-sm-ingress":
			subResources = kuberesource.Emojivoto(kuberesource.ServiceMeshIngressEgress)
			subResources = kuberesource.PatchRuntimeHandlers(subResources, runtimeHandler)
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
	kuberesource.PatchNamespaces(resources, *namespace)
	if *addNamespaceObject && *namespace != "default" && *namespace != "" {
		resources = append([]any{kuberesource.Namespace(*namespace)}, resources...)
	}

	b, err := kuberesource.EncodeResources(resources...)
	if err != nil {
		log.Fatalf("Error encoding resources: %v", err)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		log.Fatalf("Error writing to stdout: %v", err)
	}
}
