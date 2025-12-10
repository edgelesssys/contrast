// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/v72/github"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ci-watch <run-url>")
		os.Exit(1)
	}

	runURL := os.Args[1]
	u, err := url.Parse(runURL)
	if err != nil {
		fmt.Printf("Invalid URL: %v\n", err)
		os.Exit(1)
	}

	// Expected format: /owner/repo/actions/runs/runID
	parts := strings.Split(u.Path, "/")
	if len(parts) < 6 {
		fmt.Println("Invalid URL format. Expected: https://github.com/owner/repo/actions/runs/runID")
		os.Exit(1)
	}

	var owner, repo, runID string
	// Find "actions" and "runs" to locate IDs
	for i, part := range parts {
		if part == "actions" && i+2 < len(parts) && parts[i+1] == "runs" {
			runID = parts[i+2]
			if i >= 2 {
				repo = parts[i-1]
				owner = parts[i-2]
			}
			break
		}
	}

	if owner == "" || repo == "" || runID == "" {
		fmt.Println("Could not parse owner, repo, or runID from URL")
		os.Exit(1)
	}

	targetDir := "logs"
	fmt.Printf("Fetching logs for %s/%s run %s to %s\n", owner, repo, runID, targetDir)

	if err := fetchGithubWorkflowRun(context.Background(), owner, repo, runID, targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done")
}

// fetchGithubWorkflowRun fetches the logs and artifacts of a GitHub Actions workflow run.
func fetchGithubWorkflowRun(ctx context.Context, owner, repo, runID, targetDir string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	runIDInt, err := strconv.ParseInt(runID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run ID %q: %w", runID, err)
	}

	client := github.NewClient(nil).WithAuthToken(token)

	// Create run directory
	runDir := filepath.Join(targetDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	var errs []error

	fmt.Println("Fetching job console logs...")
	if err := fetchJobConsoleLogs(ctx, client, owner, repo, runIDInt, runDir); err != nil {
		errs = append(errs, fmt.Errorf("failed to fetch job logs: %w", err))
		fmt.Fprintf(os.Stderr, "Error fetching job logs: %v\n", err)
	}

	fmt.Println("Fetching artifacts...")
	if err := fetchRunArtifacts(ctx, client, owner, repo, runIDInt, runDir); err != nil {
		errs = append(errs, fmt.Errorf("failed to fetch artifacts: %w", err))
		fmt.Fprintf(os.Stderr, "Error fetching artifacts: %v\n", err)
	}

	return errors.Join(errs...)
}

func fetchJobConsoleLogs(ctx context.Context, client *github.Client, owner, repo string, runID int64, runDir string) error {
	opts := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allJobs []*github.WorkflowJob
	for {
		jobs, resp, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
		if err != nil {
			return fmt.Errorf("failed to list workflow jobs: %w", err)
		}
		allJobs = append(allJobs, jobs.Jobs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Printf("Found %d jobs\n", len(allJobs))

	var errs []error
	for _, job := range allJobs {
		if err := processJobLog(ctx, client, owner, repo, job, runDir); err != nil {
			errs = append(errs, err)
			fmt.Fprintf(os.Stderr, "Failed to process job log: %v\n", err)
		}
	}
	return errors.Join(errs...)
}

func processJobLog(ctx context.Context, client *github.Client, owner, repo string, job *github.WorkflowJob, runDir string) error {
	if job.ID == nil {
		return nil
	}

	jobName := job.GetName()
	if jobName == "" {
		jobName = fmt.Sprintf("job-%d", *job.ID)
	}

	// Sanitize job name for file path
	jobName = sanitizeName(jobName)

	jobDir := filepath.Join(runDir, jobName)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return fmt.Errorf("failed to create job directory: %w", err)
	}

	logsURL, _, err := client.Actions.GetWorkflowJobLogs(ctx, owner, repo, *job.ID, 10)
	if err != nil {
		return fmt.Errorf("getting logs URL for job %s: %w", jobName, err)
	}
	if logsURL == nil {
		fmt.Printf("Logs URL is nil for job %s (ID: %d)\n", jobName, *job.ID)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logsURL.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request for logs: %w", err)
	}

	logResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch logs for job %s: %w", jobName, err)
	}
	defer logResp.Body.Close()

	if logResp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %s", logResp.Status)
	}

	logFile := filepath.Join(jobDir, "log.txt")
	f, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, logResp.Body); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}
	return nil
}

func fetchRunArtifacts(ctx context.Context, client *github.Client, owner, repo string, runID int64, runDir string) error {
	opts := &github.ListOptions{PerPage: 100}
	var allArtifacts []*github.Artifact
	for {
		artifacts, resp, err := client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, opts)
		if err != nil {
			return fmt.Errorf("listing artifacts: %w", err)
		}
		allArtifacts = append(allArtifacts, artifacts.Artifacts...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	fmt.Printf("Found %d artifacts\n", len(allArtifacts))

	var errs []error
	for _, art := range allArtifacts {
		if err := processArtifact(ctx, client, owner, repo, art, runDir); err != nil {
			errs = append(errs, err)
			fmt.Fprintf(os.Stderr, "Failed to process artifact %s: %v\n", *art.Name, err)
		}
	}
	return errors.Join(errs...)
}

func processArtifact(ctx context.Context, client *github.Client, owner, repo string, art *github.Artifact, runDir string) error {
	if art.ID == nil || art.Name == nil {
		return nil
	}

	artName := *art.Name
	// Strip "e2e_pod_logs-" prefix to match job directories
	artName = strings.TrimPrefix(artName, "e2e_pod_logs-")
	artName = sanitizeName(artName)

	// Create dir for artifact
	// We use the artifact name as the directory name within the run directory.
	artDir := filepath.Join(runDir, artName)
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	url, _, err := client.Actions.DownloadArtifact(ctx, owner, repo, *art.ID, 10)
	if err != nil {
		return fmt.Errorf("getting download URL for artifact: %w", err)
	}
	if url == nil {
		return fmt.Errorf("download URL is nil")
	}

	if err := downloadAndExtractZip(ctx, url.String(), artDir); err != nil {
		return fmt.Errorf("downloading/extracting zip: %w", err)
	}

	fmt.Printf("Downloaded and extracted artifact: %s\n", *art.Name)
	return nil
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, " (with ", "-") // TODO(katexochen): remove with from job names
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return name
}

func downloadAndExtractZip(ctx context.Context, downloadURL, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// Read body to memory (artifacts shouldn't be massive for logs, usually)
	// If they are large, we should use a temp file.
	// Given they are logs, let's use a temp file to be safe.
	tmpFile, err := os.CreateTemp("", "artifact-*.zip")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}

	// Open zip
	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for Zip Slip
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("creating directory for file: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("opening zip file content: %w", err)
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("writing file content: %w", err)
		}
	}

	return nil
}
