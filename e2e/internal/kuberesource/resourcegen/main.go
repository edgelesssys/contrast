// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/edgelesssys/contrast/e2e/internal/kuberesource"
)

func main() {
	imageReplacementsPath := flag.String("image-replacements", "", "Path to the image replacements file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <set> <dest>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	set := flag.Arg(0)
	dest := flag.Arg(1)

	var resources []any
	var err error
	switch set {
	case "coordinator-release":
		resources, err = kuberesource.CoordinatorRelease()
	case "runtime":
		resources, err = kuberesource.Runtime()
	case "simple":
		resources, err = kuberesource.Simple()
	case "openssl":
		resources, err = kuberesource.OpenSSL()
	case "emojivoto":
		resources, err = kuberesource.Emojivoto(kuberesource.ServiceMeshDisabled)
	default:
		fmt.Printf("Error: unknown set: %s\n", set)
		os.Exit(1)
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
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

	b, err := kuberesource.EncodeResources(resources...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(path.Dir(dest), 0o755); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := os.WriteFile(dest, b, 0o644); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
