// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package release

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/google/go-github/v72/github"
	ksync "github.com/katexochen/sync/api/client"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	tokenEnvVar = "GH_TOKEN"
)

var (
	owner       = flag.String("owner", "edgelesssys", "Github repository owner")
	repo        = flag.String("repo", "contrast", "Github repository")
	tag         = flag.String("tag", "", "tag name of the release to download")
	keep        = flag.Bool("keep", false, "don't delete test resources and deployment")
	platformStr = flag.String("platform", "", "Deployment platform")
)

// TestRelease downloads a release from Github, sets up the coordinator, installs the demo
// deployment and runs some simple smoke tests.
func TestRelease(t *testing.T) {
	ctx := t.Context()
	k := kubeclient.NewForTest(t)

	lowerPlatformStr := strings.ToLower(*platformStr)
	// On AKS, wait for a load balancer, on bare-metal connect directly to the cluster IP.
	hasLoadBalancer := strings.HasPrefix(lowerPlatformStr, "aks-")

	dir := fetchRelease(ctx, t)

	contrast := &contrast{dir: dir}

	// If available, acquire a fifo ticket to synchronize cluster access with
	// other running e2e tests. We request a ticket and wait for our turn.
	// Ticket is released in the cleanup function. The sync server will ensure
	// that only one test is using the cluster at a time.
	var fifo *ksync.Fifo
	if fifoUUID, ok := os.LookupEnv("SYNC_FIFO_UUID"); ok {
		syncEndpoint, ok := os.LookupEnv("SYNC_ENDPOINT")
		require.True(t, ok, "SYNC_ENDPOINT must be set when SYNC_FIFO_UUID is set")
		t.Logf("Syncing with fifo %s of endpoint %s", fifoUUID, syncEndpoint)
		fifo = ksync.FifoFromUUID(syncEndpoint, fifoUUID)
		err := fifo.TicketAndWait(ctx)
		if err != nil {
			t.Log("If this throws a 404, likely the sync server was restarted.")
			t.Log("Run 'nix run .#scripts.renew-sync-fifo' against the CI cluster to fix it.")
			require.NoError(t, err)
		}
		t.Logf("Acquired lock on fifo %s", fifoUUID)
	}

	for _, sub := range []string{"help"} {
		contrast.Run(ctx, t, 2*time.Second, sub)
	}

	t.Cleanup(func() {
		ctx := context.Background() //nolint:usetesting // see https://github.com/ldez/usetesting/issues/4
		defer func() {
			if fifo != nil {
				if err := fifo.Done(ctx); err != nil {
					t.Logf("Could not mark fifo ticket as done: %v", err)
				}
			}
		}()

		if *keep {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		var resources []*unstructured.Unstructured
		for _, subdir := range []string{".", "deployment"} {
			files, err := filepath.Glob(filepath.Join(dir, subdir, "*.yml"))
			if err != nil {
				// err is a bad glob pattern, that should not happen!
				panic(err)
			}
			for _, file := range files {
				t.Logf("reading %q", file)
				yaml, err := os.ReadFile(file)
				require.NoError(t, err)
				rs, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
				require.NoError(t, err)
				resources = append(resources, rs...)
			}
		}

		// Delete resources 1-by-1 so that we don't stop on errors.
		for _, resource := range resources {
			if err := k.Delete(ctx, resource); err != nil {
				t.Logf("deleting resource %s: %v", resource.GetName(), err)
			}
		}
	})

	require.True(t, t.Run("apply-runtime", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		yaml, err := os.ReadFile(path.Join(dir, fmt.Sprintf("runtime-%s.yml", lowerPlatformStr)))
		require.NoError(err)
		resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
		require.NoError(err)

		require.NoError(k.Apply(ctx, resources...))

		for _, r := range resources {
			if r.GetKind() != "DaemonSet" {
				continue
			}
			require.NoError(k.WaitForDaemonSet(ctx, r.GetNamespace(), r.GetName()))
		}
	}), "the runtime is required for subsequent tests to run")

	require.True(t, t.Run("process-deployment", func(t *testing.T) {
		require := require.New(t)

		require.NoError(os.Mkdir(path.Join(dir, "deployment"), 0o777))
		require.NoError(os.Rename(path.Join(dir, "emojivoto-demo.yml"), path.Join(dir, "deployment", "emojivoto-demo.yml")))
		require.NoError(os.Rename(path.Join(dir, "coordinator.yml"), path.Join(dir, "deployment", "coordinator.yml")))

		infos, err := os.ReadDir(path.Join(dir, "deployment"))
		require.NoError(err)
		for _, info := range infos {
			name := path.Join(path.Join(dir, "deployment"), info.Name())
			yaml, err := os.ReadFile(name)
			require.NoError(err)
			resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
			require.NoError(err)

			newYAML, err := kuberesource.EncodeUnstructured(resources)
			require.NoError(err)
			require.NoError(os.WriteFile(name, newYAML, 0o644))

		}
	}), "deployment processing needs to succeed for subsequent tests to run")

	contrast.Run(ctx, t, 4*time.Minute, "generate", "--reference-values", *platformStr, "deployment/")

	require.True(t, t.Run("patch reference values", func(t *testing.T) {
		require := require.New(t)

		platf, err := platforms.FromString(lowerPlatformStr)
		require.NoError(err)

		manifestBytes, err := os.ReadFile(contrast.dir + "/manifest.json")
		require.NoError(err)
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))

		patchManifest, err := contrasttest.PatchReferenceValues(t.Context(), k, platf)
		require.NoError(err)
		m = patchManifest(m)

		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(contrast.dir+"/manifest.json", manifestBytes, 0o644))
	}), "patching reference values needs to succeed for subsequent tests to run")

	require.True(t, t.Run("apply-demo", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		files, err := filepath.Glob(path.Join(dir, "deployment", "*.yml"))
		require.NoError(err)
		for _, file := range files {
			yaml, err := os.ReadFile(file)
			require.NoError(err)
			resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
			require.NoError(err)
			require.NoError(k.Apply(ctx, resources...))
		}
	}), "applying the demo is required for subsequent tests to run")

	var coordinatorIP string
	require.True(t, t.Run("wait-for-coordinator-ip", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		require.NoError(k.WaitForCoordinator(ctx, "default"))
		var err error
		coordinatorIP, err = k.WaitForService(ctx, "default", "coordinator", hasLoadBalancer)
		require.NoError(err)
	}), "the coordinator is required for subsequent tests to run")

	setFlags := []string{"set", "-c", coordinatorIP + ":1313", "deployment/"}
	contrast.Run(ctx, t, 1*time.Minute, setFlags...)

	require.True(t, t.Run("wait-for-deployment-readiness", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		require.NoError(k.WaitForDeployment(ctx, "default", "vote-bot"))
		require.NoError(k.WaitForDeployment(ctx, "default", "voting"))
		require.NoError(k.WaitForDeployment(ctx, "default", "emoji"))
		require.NoError(k.WaitForDeployment(ctx, "default", "web"))
	}), "the deployment must get ready for subsequent tests to run")

	verifyFlags := []string{"verify", "-c", coordinatorIP + ":1313"}
	contrast.Run(ctx, t, 1*time.Minute, verifyFlags...)

	t.Run("test-demo", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		emojiwebIP, err := k.WaitForService(ctx, "default", "web-svc", hasLoadBalancer)
		require.NoError(err)

		cfg := &tls.Config{RootCAs: x509.NewCertPool()}
		pem, err := os.ReadFile(path.Join(dir, "verify", "mesh-ca.pem"))
		require.NoError(err)
		require.True(cfg.RootCAs.AppendCertsFromPEM(pem))

		c := http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(emojiwebIP, "443"))
				},
				TLSClientConfig: cfg,
			},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://web", nil)
		require.NoError(err)
		resp, err := c.Do(req)
		require.NoError(err)
		defer resp.Body.Close()
		require.Equal(http.StatusOK, resp.StatusCode)
	})
}

type contrast struct {
	dir     string
	binName string
}

func (c *contrast) Run(ctx context.Context, t *testing.T, timeout time.Duration, args ...string) {
	require.True(t, t.Run(args[0], func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if c.binName == "" {
			c.binName = "contrast"
		}
		args := append([]string{"--log-level", "debug"}, args...)
		cmd := exec.CommandContext(ctx, fmt.Sprintf("./%s", c.binName), args...)
		cmd.Dir = c.dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "output:\n%s", string(out))
	}), args[0]+" needs to succeed for subsequent tests to run")
}

// fetchRelease downloads the release corresponding to the global tag variable and returns the directory.
func fetchRelease(ctx context.Context, t *testing.T) string {
	require := require.New(t)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	token := os.Getenv(tokenEnvVar)
	require.NotEmpty(token, "environment variable %q must contain a Github access token", tokenEnvVar)
	gh := github.NewClient(nil).WithAuthToken(token)

	var dir string
	if *keep {
		var err error
		dir, err = os.MkdirTemp("", "releasetest-")
		require.NoError(err)
		t.Logf("Created test directory %s", dir)
	} else {
		dir = t.TempDir()
	}

	// Find our target release. There is GetReleaseByTag, but we may be looking for a draft release.
	rels, resp, err := gh.Repositories.ListReleases(ctx, *owner, *repo, nil)
	require.NoError(err)
	var release *github.RepositoryRelease
	for _, rel := range rels {
		t.Logf("Checking release %q", *rel.TagName)
		if *rel.TagName == *tag {
			release = rel
			break
		}
	}
	require.NotNil(release, "release %q not found among %d releases\nGithub response:\n%#v", *tag, len(rels), resp)

	for _, asset := range release.Assets {
		f, err := os.OpenFile(path.Join(dir, *asset.Name), os.O_CREATE|os.O_RDWR, 0o777)
		require.NoError(err)
		body, _, err := gh.Repositories.DownloadReleaseAsset(ctx, *owner, *repo, *asset.ID, http.DefaultClient)
		require.NoError(err, "could not fetch release asset %q (id: %d)", asset.Name, asset.ID)
		_, err = io.Copy(f, body)
		require.NoError(err)
		f.Close()
	}

	return dir
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
