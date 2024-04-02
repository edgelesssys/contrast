package main

import (
	"fmt"
	"os"
	"path"

	"github.com/edgelesssys/contrast/e2e/internal/kuberesource"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: kuberesource <set> <dest>")
		os.Exit(1)
	}

	set := os.Args[1]
	dest := os.Args[2]

	var resources []any
	var err error
	switch set {
	case "simple":
		resources, err = kuberesource.Simple()
	case "openssl":
		resources, err = kuberesource.OpenSSL()
	case "emojivoto":
		resources, err = kuberesource.Emojivoto()
	default:
		fmt.Printf("Error: unknown set: %s\n", set)
		os.Exit(1)
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

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
