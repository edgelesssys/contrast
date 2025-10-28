// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package genpolicy

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/edgelesssys/contrast/internal/embedbin"
	"github.com/edgelesssys/contrast/internal/kuberesource"
)

// Runner is a wrapper around the genpolicy tool.
//
// Create an instance with New(), call Run() to execute the tool, and call
// Teardown() afterwards to clean up temporary files.
type Runner struct {
	genpolicy embedbin.Installed

	rulesPath    string
	settingsPath string
	cachePath    string
}

// New creates a new Runner for the given configuration.
func New(rulesPath, settingsPath, cachePath string, bin []byte) (*Runner, error) {
	e := embedbin.New()
	genpolicy, err := e.Install("", bin)
	if err != nil {
		return nil, fmt.Errorf("installing genpolicy: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o777); err != nil {
		return nil, fmt.Errorf("creating cache file: %w", err)
	}

	runner := &Runner{
		genpolicy:    genpolicy,
		rulesPath:    rulesPath,
		settingsPath: settingsPath,
		cachePath:    cachePath,
	}

	return runner, nil
}

// Run runs the tool on the given resource and returns the initdata annotation.
//
// Run can be called more than once.
func (r *Runner) Run(ctx context.Context, res any, extraPath string, logger *slog.Logger) (string, error) {
	args := []string{
		"--runtime-class-names=contrast-cc",
		"--rego-rules-path=" + r.rulesPath,
		"--json-settings-path=" + r.settingsPath,
		"--layers-cache-file-path=" + r.cachePath,
		"--yaml-file=/dev/stdin", // prevent genpolicy from outputting anything but the annotation
		"--config-file=" + extraPath,
		"--base64-out",
	}
	genpolicy := exec.CommandContext(ctx, r.genpolicy.Path(), args...)
	input, err := kuberesource.EncodeResources(res)
	if err != nil {
		return "", fmt.Errorf("encoding resources: %w", err)
	}
	genpolicy.Stdin = bytes.NewReader(input)
	genpolicy.Env = os.Environ()
	if _, hasRustLog := os.LookupEnv("RUST_LOG"); !hasRustLog {
		genpolicy.Env = append(genpolicy.Env, "RUST_LOG=info")
	}
	if _, hasRustBacktrace := os.LookupEnv("RUST_BACKTRACE"); !hasRustBacktrace {
		genpolicy.Env = append(genpolicy.Env, "RUST_BACKTRACE=1")
	}

	logFilter := newLogTranslator(logger)
	defer logFilter.stop()

	var out bytes.Buffer
	genpolicy.Stdout = &out
	genpolicy.Stderr = logFilter

	logger.Debug("running genpolicy", "bin", r.genpolicy.Path(), "args", args)
	if err := genpolicy.Run(); err != nil {
		return "", fmt.Errorf("running genpolicy: %w", err)
	}

	return out.String(), err
}

// Teardown cleans up temporary files and should be called after the last Run.
func (r *Runner) Teardown() error {
	if r.genpolicy != nil {
		return r.genpolicy.Uninstall()
	}
	return nil
}
