// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func fixBomrefsCmd(args []string) error {
	if len(args) != 1 {
		return errors.New("fix-bomrefs: expected exactly one file argument")
	}
	path := args[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	walk(root, func(n map[string]any) {
		if ref, ok := n["bom-ref"].(string); ok {
			seen[ref] = struct{}{}
		}
	})
	walk(root, func(n map[string]any) {
		if needsRef(n) {
			n["bom-ref"] = uniqueRef(componentRef(n), seen)
		}
	})
	return writeJSON(path, root)
}

func walk(node any, fn func(map[string]any)) {
	switch n := node.(type) {
	case map[string]any:
		fn(n)
		for _, v := range n {
			walk(v, fn)
		}
	case []any:
		for _, v := range n {
			walk(v, fn)
		}
	}
}

func needsRef(n map[string]any) bool {
	if _, ok := n["type"].(string); !ok {
		return false
	}
	if _, ok := n["name"].(string); !ok {
		return false
	}
	_, hasRef := n["bom-ref"].(string)
	return !hasRef
}

func componentRef(n map[string]any) string {
	if purl, ok := n["purl"].(string); ok && purl != "" {
		return purl
	}
	name, _ := n["name"].(string)
	version, _ := n["version"].(string)
	return name + "@" + version
}

func uniqueRef(base string, seen map[string]struct{}) string {
	ref := base
	for i := 2; ; i++ {
		if _, clash := seen[ref]; !clash {
			seen[ref] = struct{}{}
			return ref
		}
		ref = fmt.Sprintf("%s#%d", base, i)
	}
}

func writeJSON(path string, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
