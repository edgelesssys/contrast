// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const cdxSchema = "http://cyclonedx.org/schema/bom-1.6.schema.json"

type bom struct {
	Schema       string      `json:"$schema"`
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber"`
	Version      int         `json:"version"`
	Metadata     *metadata   `json:"metadata,omitempty"`
	Components   []component `json:"components,omitempty"`
}

type metadata struct {
	Component *component `json:"component,omitempty"`
}

type component struct {
	Type       string     `json:"type"`
	BOMRef     string     `json:"bom-ref"`
	Group      string     `json:"group,omitempty"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	PURL       string     `json:"purl,omitempty"`
	Properties []property `json:"properties,omitempty"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

var storePathRE = regexp.MustCompile(`^[0-9a-z]{32}-(.+?)(?:-([0-9][^-/]*))?$`)

func closureCmd(args []string) error {
	fs := flag.NewFlagSet("closure", flag.ContinueOnError)
	storePaths := fs.String("store-paths", "", "path to closureInfo store-paths file")
	rootVersion := fs.String("version", "", "version of the root component")
	output := fs.String("output", "", "output CycloneDX JSON path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *storePaths == "" || *output == "" {
		return errors.New("closure: --store-paths and --output are required")
	}

	paths, err := readPaths(*storePaths)
	if err != nil {
		return err
	}

	components := make([]component, 0, len(paths))
	for _, p := range paths {
		name, version, ref := parseStorePath(p)
		components = append(components, component{
			Type:    "library",
			BOMRef:  ref,
			Name:    name,
			Version: version,
			PURL:    fmt.Sprintf("pkg:nix/%s@%s", name, version),
			Properties: []property{
				{Name: "nix:store_path", Value: p},
			},
		})
	}

	return writeJSON(*output, bom{
		Schema:       cdxSchema,
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.6",
		SerialNumber: deterministicURN(paths, *rootVersion),
		Version:      1,
		Metadata: &metadata{
			Component: &component{
				Type:    "application",
				BOMRef:  "contrast@" + *rootVersion,
				Group:   "io.edgeless",
				Name:    "contrast",
				Version: *rootVersion,
			},
		},
		Components: components,
	})
}

func parseStorePath(path string) (name, version, ref string) {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	m := storePathRE.FindStringSubmatch(base)
	if m == nil {
		return base, "0.0.0", base
	}
	version = m[2]
	if version == "" {
		version = "0.0.0"
	}
	return m[1], version, base
}

// readPaths sorts store paths so output is stable across runs.
func readPaths(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	seen := make(map[string]struct{})
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		seen[line] = struct{}{}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// deterministicURN returns an RFC 4122 v4 UUID derived deterministically.
func deterministicURN(paths []string, rootVersion string) string {
	h := sha256.New()
	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte{'\n'})
	}
	h.Write([]byte(rootVersion))
	sum := h.Sum(nil)[:16]
	sum[6] = (sum[6] & 0x0f) | 0x40 // version 4
	sum[8] = (sum[8] & 0x3f) | 0x80 // RFC 4122 variant
	hexed := hex.EncodeToString(sum)
	return fmt.Sprintf("urn:uuid:%s-%s-%s-%s-%s",
		hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}
