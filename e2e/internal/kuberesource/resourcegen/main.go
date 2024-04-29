// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/edgelesssys/contrast/e2e/internal/kuberesource"
)

func main() {
	imageReplacementsPath := flag.String("image-replacements", "", "Path to the image replacements file")
	namespace := flag.String("namespace", "", "Namespace for namespaced resources")
	addNamespaceObject := flag.Bool("add-namespace-object", false, "Add namespace object with the given namespace")
	addPortForwarders := flag.Bool("add-port-forwarders", false, "Add port forwarder pods for all services")
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
			c := kuberesource.Coordinator("").DeploymentApplyConfiguration
			s := kuberesource.ServiceForDeployment(c)
			subResources, err = []any{c, s}, nil
		case "coordinator-release":
			subResources, err = kuberesource.CoordinatorRelease()
		case "runtime":
			subResources, err = kuberesource.Runtime()
		case "openssl":
			subResources, err = kuberesource.OpenSSL()
		case "emojivoto":
			subResources, err = kuberesource.Emojivoto(kuberesource.ServiceMeshDisabled)
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
