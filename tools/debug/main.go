// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	_ "github.com/edgelesssys/contrast/internal/attestation/tdx"
	"github.com/edgelesssys/contrast/tools/debug/runners"
)

var (
	addr         = flag.String("addr", "", "coordinator address")
	n            = flag.Uint("num-runs", 10, "number of evaluations")
	manifestFile = flag.String("manifest", "", "Path to manifest file")
	benchmarks   = flag.String("benchmarks", "tcp,atls-issue,atls-verify", "comma-separated list of benchmarks to run")
	cache        = flag.String("cache-dir", "", "where to store cached files (default: temp dir)")
)

func main() {
	flag.Parse()

	names := strings.Split(*benchmarks, ",")

	verifyingATLS, err := runners.NewATLSWithValidators(*addr, *manifestFile)
	if err != nil {
		log.Fatalf("Creating aTLS validators: %v", err)
	}
	cases := map[string]Runner{
		"tcp":         runners.NewTCP(*addr),
		"tls":         runners.NewTLS(*addr),
		"atls-issue":  runners.NewATLS(*addr),
		"atls-verify": verifyingATLS,
	}

	encoder := json.NewEncoder(os.Stdout)

	for name, c := range cases {
		if !slices.Contains(names, name) {
			continue
		}
		timings, err := benchmark(context.Background(), 10, c)
		if err != nil {
			log.Fatalf("Benchmark %q failed: %v", name, err)
		}
		if err := encoder.Encode(process(name, timings)); err != nil {
			log.Fatalf("Encoding result of %q: %v", name, err)
		}
	}
}

type Runner interface {
	Run(context.Context) error
}

func benchmark(ctx context.Context, numRuns int, runner Runner) ([]time.Duration, error) {
	var results []time.Duration

	for i := range numRuns {
		t := time.Now()
		if err := runner.Run(ctx); err != nil {
			return nil, fmt.Errorf("error in run %d: %w", i, err)
		}
		results = append(results, time.Since(t))
	}

	return results, nil
}

type result struct {
	Name    string          `json:"name"`
	Timings []time.Duration `json:"timings"`
	Median  string          `json:"median"`
}

func process(name string, timings []time.Duration) *result {
	k := len(timings) / 2
	median := timings[k]
	if len(timings)%2 == 1 {
		median = (median + timings[k+1]) / 2
	}

	return &result{Name: name, Timings: timings, Median: median.String()}
}
