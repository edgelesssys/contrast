// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kconfig

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Config represents a Linux kernel configuration.
type Config struct {
	lines []string
	index map[string]int
}

// Parse parses a kernel config from a reader.
func Parse(r io.Reader) (*Config, error) {
	c := &Config{
		index: make(map[string]int),
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		c.lines = append(c.lines, line)
		key, err := parseKey(line)
		if err != nil {
			return nil, err
		}
		if key == "" {
			continue
		}
		_, ok := c.index[key]
		if ok {
			return nil, fmt.Errorf("encountered a duplicate key: %q", key)
		}

		c.index[key] = len(c.lines) - 1

	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return c, nil
}

func parseKey(line string) (string, error) {
	reSet := regexp.MustCompile(`^(CONFIG_[A-Za-z0-9_]+)=`)
	if m := reSet.FindStringSubmatch(line); m != nil {
		return m[1], nil
	}

	reUnset := regexp.MustCompile(`^# (CONFIG_[A-Za-z0-9_]+) is not set$`)
	if m := reUnset.FindStringSubmatch(line); m != nil {
		return m[1], nil
	}

	// Ignore empty and comment lines.
	if line == "" || strings.HasPrefix(line, "#") {
		return "", nil
	}

	return "", fmt.Errorf("invalid config line format: %q", line)
}

// Set sets a configuration option.
func (c *Config) Set(key, value string) {
	line := fmt.Sprintf("%s=%s", key, value)
	if idx, ok := c.index[key]; ok {
		c.lines[idx] = line
	} else {
		c.lines = append(c.lines, line)
		c.index[key] = len(c.lines) - 1
	}
}

// Unset unsets a configuration option.
func (c *Config) Unset(key string) {
	line := fmt.Sprintf("# %s is not set", key)
	if idx, ok := c.index[key]; ok {
		c.lines[idx] = line
	} else {
		c.lines = append(c.lines, line)
		c.index[key] = len(c.lines) - 1
	}
}

// Marshal serializes the config to bytes.
func (c *Config) Marshal() []byte {
	var buf bytes.Buffer
	for _, line := range c.lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}
