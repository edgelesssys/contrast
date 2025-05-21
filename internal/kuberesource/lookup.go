// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

var replacementRE = regexp.MustCompile(`(?P<image>[^\s=]+)\s*=\s*(?P<replacement>\S+)`)

// ImageReplacementsFromFile parses the containerlookup file into a map.
//
// The file is expected to contain newline-separated pairs of images and their intended
// replacement, separated by an = sign. Empty lines and lines starting with the pound character
// are ignored. This file is populated by container image build rules in the justfile.
func ImageReplacementsFromFile(file io.Reader) (map[string]string, error) {
	m := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		matches := replacementRE.FindStringSubmatch(line)
		if matches == nil {
			return nil, fmt.Errorf("invalid image line: %s", line)
		}

		if replacementRE.SubexpIndex("image") == -1 {
			return nil, fmt.Errorf("image not found for image line: %s", line)
		}
		image := matches[replacementRE.SubexpIndex("image")]

		if replacementRE.SubexpIndex("replacement") == -1 {
			return nil, fmt.Errorf("replacement not found for image line: %s", line)
		}
		replacement := matches[replacementRE.SubexpIndex("replacement")]

		m[image] = replacement
	}

	return m, nil
}
