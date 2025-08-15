// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/edgelesssys/contrast/imagepuller/client"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/spf13/cobra"
)

const (
	imagepullerDir  = "tmp_imagepuller"
	maxPullDuration = 10 * time.Minute
)

var imageList = []string{
	"ghcr.io/edgelesssys/contrast/dmesg:v0.0.1@sha256:6ad6bbb5735b84b10af42d2441e8d686b1d9a6cbf096b53842711ef5ddabd28d",
	"ghcr.io/edgelesssys/contrast/coordinator@sha256:6f966a922cc9a39d7047ed41ffafc7eb7a3c6a4fd8966cbf30fa902b455789f7",
	"tensorflow/tensorflow:latest-gpu@sha256:73fe35b67dad5fa5ab0824ed7efeb586820317566a705dff76142f8949ffcaff",
}
var mountPoint = "current_server"

func getDiskUsage(path string) (uint64, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return 0, err
	}
	return usage.Used, nil
}

func extractName(name string) string {
	at := strings.Index(name, "@")
	if at == -1 {
		return ""
	}
	slash := strings.LastIndex(name[:at], "/")
	return name[slash+1 : at]
}

func cleanup(storagePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "findmnt", "-rn", "-o", "TARGET")
	output, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("findmnt returned non-zero exit code, stderr is: %s", string(exitErr.Stderr))
	} else if err != nil {
		return fmt.Errorf("failed to execute findmnt: %w", err)
	}

	var mountpoints []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, storagePath) {
			mountpoints = append(mountpoints, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading findmnt output: %w", err)
	}

	sort.Slice(mountpoints, func(i, j int) bool {
		return strings.Count(mountpoints[i], "/") > strings.Count(mountpoints[j], "/")
	})

	for _, mp := range mountpoints {
		if err := unix.Unmount(mp, 0); err != nil {
			return fmt.Errorf("unmounting %s: %w", mp, err)
		}
	}

	if err := os.RemoveAll(storagePath); err != nil {
		return fmt.Errorf("removing directory %s: %w", storagePath, err)
	}
	return nil
}

func findChildPid(ctx context.Context, ppid int) (int, error) {
	out, err := exec.CommandContext(ctx, "ps", "-o", "pid=", "--ppid", fmt.Sprint(ppid)).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return 0, fmt.Errorf("ps returned non-zero exit code, stderr is: %s", string(exitErr.Stderr))
	} else if err != nil {
		return 0, fmt.Errorf("failed to execute ps: %w", err)
	}

	lines := strings.SplitSeq(string(out), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		return pid, nil
	}
	return 0, fmt.Errorf("no child found for PID %d", ppid)
}

func startServerWithMemoryTracking(ctx context.Context, serverPath string, args ...string) (func() (int, error), int, error) {
	var cmd *exec.Cmd

	timeCmd, err := exec.LookPath("time")
	if err != nil {
		return nil, 0, fmt.Errorf("\"time\" is not in PATH: %w", err)
	}

	argsFull := append([]string{"-v", serverPath}, args...)
	cmd = exec.CommandContext(ctx, timeCmd, argsFull...)
	cmd.Stdout = nil
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("listening on stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, 0, fmt.Errorf("failed to start server: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	childPid, err := findChildPid(ctx, cmd.Process.Pid)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find server child PID: %w", err)
	}

	// Closure that will wait and extract MaxRSS after process exit
	waitAndGetMaxRSS := func() (int, error) {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, "Maximum resident set size (kbytes)") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 6 {
				continue
			}
			kb, err := strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return 0, fmt.Errorf("parsing MaxRSS failed: %w", err)
			}
			return kb, nil
		}

		waitErr := cmd.Wait()
		if waitErr != nil {
			return 0, fmt.Errorf("command exited with error: %w", waitErr)
		}
		return 0, fmt.Errorf("MaxRSS not found in output")
	}

	return waitAndGetMaxRSS, childPid, nil
}

func profileServerIndividual(serverPath, args, storagePath string, label string) (_ map[string]resourceUsage, retErr error) {
	fmt.Printf("===== Testing server (individual): %s =====\n", label)
	defer func() {
		if err := errors.Join(cleanup(storagePath), cleanup(mountPoint)); err != nil {
			retErr = err
		}
	}()

	results := map[string]resourceUsage{}
	for _, image := range imageList {
		if err := errors.Join(cleanup(storagePath), cleanup(mountPoint)); err != nil {
			return nil, err
		}
		fmt.Printf("[%s]\n", extractName(image))

		ctx, cancel := context.WithTimeout(context.Background(), maxPullDuration)
		defer cancel()

		waitForRSS, childPid, err := startServerWithMemoryTracking(ctx, serverPath, args)
		if err != nil {
			return nil, err
		}
		time.Sleep(500 * time.Millisecond)

		diskBefore, err := getDiskUsage(storagePath)
		if err != nil {
			return nil, err
		}

		start := time.Now()
		if err := client.Request(image, mountPoint, maxPullDuration); err != nil {
			return nil, err
		}

		duration := time.Since(start)
		if err := syscall.Kill(childPid, syscall.SIGKILL); err != nil {
			return nil, err
		}
		diskAfter, err := getDiskUsage(storagePath)
		if err != nil {
			return nil, err
		}
		maxRSSkb, err := waitForRSS()
		if err != nil {
			return nil, err
		}

		result := resourceUsage{
			Time:    int(duration.Seconds()),
			Memory:  maxRSSkb / 1024,
			Storage: int(diskAfter-diskBefore) / 1024 / 1024,
		}
		results[fmt.Sprintf("%s-%s", extractName(image), label)] = result
		fmt.Printf("Time taken: %d s\n", result.Time)
		fmt.Printf("Memory peak: %d MB\n", result.Memory)
		fmt.Printf("Storage used: %d MB\n", result.Storage)
		fmt.Println()
	}
	return results, nil
}

func profileServerContinuous(serverPath, args, storagePath string, label string) (_ resourceUsage, retErr error) {
	fmt.Printf("===== Testing server (continuous): %s =====\n", label)
	if err := cleanup(storagePath); err != nil {
		return resourceUsage{}, err
	}
	defer func() {
		if err := errors.Join(cleanup(storagePath), cleanup(mountPoint)); err != nil {
			retErr = err
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), maxPullDuration)
	defer cancel()

	waitForRSS, childPid, err := startServerWithMemoryTracking(ctx, serverPath, args)
	if err != nil {
		return resourceUsage{}, err
	}
	time.Sleep(500 * time.Millisecond)

	diskBefore, err := getDiskUsage(storagePath)
	if err != nil {
		return resourceUsage{}, err
	}

	start := time.Now()
	for _, image := range imageList {
		if err := cleanup(mountPoint); err != nil {
			return resourceUsage{}, err
		}
		err = client.Request(image, mountPoint, maxPullDuration)
		if err != nil {
			return resourceUsage{}, err
		}
	}
	duration := time.Since(start)

	if err := syscall.Kill(childPid, syscall.SIGKILL); err != nil {
		return resourceUsage{}, err
	}
	diskAfter, err := getDiskUsage(storagePath)
	if err != nil {
		return resourceUsage{}, err
	}
	maxRSSkb, err := waitForRSS()
	if err != nil {
		return resourceUsage{}, err
	}

	result := resourceUsage{
		Time:    int(duration.Seconds()),
		Memory:  maxRSSkb / 1024,
		Storage: int(diskAfter-diskBefore) / 1024 / 1024,
	}
	fmt.Printf("Time taken: %d s\n", result.Time)
	fmt.Printf("Memory peak: %d MB\n", result.Memory)
	fmt.Printf("Storage used: %d MB\n", result.Storage)
	fmt.Println()
	return result, nil
}

func compareResourceUsage(baselineFile string, data map[string]resourceUsage, threshold float64, delta int) error {
	baselineRaw, err := os.ReadFile(baselineFile)
	if err != nil {
		return fmt.Errorf("failed to read compare file: %w", err)
	}

	var baseline map[string]resourceUsage
	if err := json.Unmarshal(baselineRaw, &baseline); err != nil {
		return fmt.Errorf("failed to parse compare file: %w", err)
	}

	var allErrs []error
	for name, curr := range data {
		prev, ok := baseline[name]
		if !ok {
			continue // skip entries not found in baseline
		}

		checkRelative := func(label string, oldVal, newVal int) {
			diff := float64(newVal-oldVal) / float64(oldVal)
			if diff > threshold { // we do not care about reductions in values
				allErrs = append(allErrs, fmt.Errorf("%s usage increased by %.1f%% for %q (was %d, now %d)", label, diff*100, name, oldVal, newVal))
			}
		}

		checkDelta := func(label string, oldVal, newVal int) {
			diff := newVal - oldVal
			if diff > delta { // we do not care about reductions in values
				fmt.Printf("WARN: %s usage increased by %d for %q (was %d, now %d)\n", label, diff, name, oldVal, newVal)
			}
		}

		checkRelative("memory", prev.Memory, curr.Memory)
		checkRelative("storage", prev.Storage, curr.Storage)
		checkDelta("time", prev.Time, curr.Time)
	}

	return errors.Join(allErrs...)
}

type resourceUsage struct {
	Time    int `json:"time"`
	Memory  int `json:"memory"`
	Storage int `json:"storage"`
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "imagepuller-benchmark <path-to-bin>",
		Short:        "benchmark imagepuller",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         run,
	}

	cmd.Flags().StringP("output", "o", "", "write result as JSON to the specified file")
	cmd.Flags().StringP("compare", "c", "", "compare against JSON file")
	cmd.Flags().Float64P("threshold", "t", 0.20, "relative threshold above which an error is thrown when comparing results")
	cmd.Flags().IntP("delta", "d", 15, "absolute time delta in seconds above which an error is thrown when comparing results")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	binPath := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	compare, err := cmd.Flags().GetString("compare")
	if err != nil {
		return err
	}
	threshold, err := cmd.Flags().GetFloat64("threshold")
	if err != nil {
		return err
	}
	delta, err := cmd.Flags().GetInt("delta")
	if err != nil {
		return err
	}

	results := map[string]resourceUsage{}
	resultsIndividual, err := profileServerIndividual(binPath, fmt.Sprintf("--tmpdir=%s", imagepullerDir), imagepullerDir, "imagepuller")
	if err != nil {
		return err
	}
	maps.Copy(results, resultsIndividual)
	resultsContinuous, err := profileServerContinuous(binPath, fmt.Sprintf("--tmpdir=%s", imagepullerDir), imagepullerDir, "imagepuller")
	if err != nil {
		return err
	}
	results["continuous"] = resultsContinuous

	if output != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(output, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}

	if compare != "" {
		return compareResourceUsage(compare, results, threshold, delta)
	}

	return nil
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
