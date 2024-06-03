// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package release

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/google/go-github/v62/github"
	"github.com/stretchr/testify/require"
)

const (
	tokenEnvVar = "GH_TOKEN"
)

var (
	owner     = flag.String("owner", "edgelesssys", "Github repository owner")
	repo      = flag.String("repo", "contrast", "Github repository")
	tag       = flag.String("tag", "", "tag name of the release to download")
	namespace = flag.String("namespace", "", "k8s namespace to install resources to (will be deleted unless --keep is set)")
	keep      = flag.Bool("keep", false, "don't delete test resources and deployment")
)

// TestRelease downloads a release from Github, sets up the coordinator, installs the demo
// deployment and runs some simple smoke tests.
func TestRelease(t *testing.T) {
	ctx := context.Background()
	k := kubeclient.NewForTest(t)

	if *namespace == "" {
		*namespace = randomNamespace(t)
		t.Logf("Created test namespace %s", *namespace)
	}

	require.True(t, t.Run("create-namespace", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		res, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(*namespace)})
		require.NoError(err)
		require.NoError(k.Apply(ctx, res...))
	}), "the namespace is required for subsequent tests to run")

	t.Cleanup(func() {
		if *keep {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		res, err := kuberesource.ResourcesToUnstructured([]any{kuberesource.Namespace(*namespace)})
		if err != nil {
			return
		}
		k.Delete(ctx, res...)
	})

	dir := fetchRelease(ctx, t)

	contrast := &contrast{dir}

	for _, sub := range []string{"help"} {
		contrast.Run(t, ctx, 2*time.Second, sub)
	}
	require.True(t, t.Run("apply-runtime", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		yaml, err := os.ReadFile(path.Join(dir, "runtime.yml"))
		require.NoError(err)
		resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
		require.NoError(err)

		for _, r := range resources {
			if r.GetKind() != "RuntimeClass" {
				r.SetNamespace(*namespace)
			}
		}

		require.NoError(k.Apply(ctx, resources...))
		require.NoError(k.WaitForDaemonset(ctx, *namespace, "contrast-node-installer"))
	}), "the runtime is required for subsequent tests to run")

	var coordinatorIP string
	require.True(t, t.Run("apply-coordinator", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		yaml, err := os.ReadFile(path.Join(dir, "coordinator.yml"))
		require.NoError(err)
		resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
		require.NoError(err)

		for _, r := range resources {
			r.SetNamespace(*namespace)
		}

		require.NoError(k.Apply(ctx, resources...))
		require.NoError(k.WaitForStatefulSet(ctx, *namespace, "coordinator"))
		coordinatorIP, err = k.WaitForLoadBalancer(ctx, *namespace, "coordinator")
		require.NoError(err)
	}), "the coordinator is required for subsequent tests to run")

	require.True(t, t.Run("unpack-deployment", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "unzip", "emojivoto-demo.zip")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(err, "output:\n%s", string(out))

		infos, err := os.ReadDir(path.Join(dir, "deployment"))
		require.NoError(err)
		for _, info := range infos {
			name := path.Join(path.Join(dir, "deployment"), info.Name())
			yaml, err := os.ReadFile(name)
			require.NoError(err)
			resources, err := kubeapi.UnmarshalUnstructuredK8SResource(yaml)
			require.NoError(err)

			for _, r := range resources {
				r.SetNamespace(*namespace)
			}
			newYAML, err := kuberesource.EncodeUnstructured(resources)
			require.NoError(err)
			require.NoError(os.WriteFile(name, newYAML, 0o644))

		}
	}), "unpacking needs to succeed for subsequent tests to run")

	contrast.Run(t, ctx, 2*time.Minute, "generate", "deployment/")
	contrast.Run(t, ctx, 1*time.Minute, "set", "-c", coordinatorIP+":1313", "deployment/")
	contrast.Run(t, ctx, 1*time.Minute, "verify", "-c", coordinatorIP+":1313")

	require.True(t, t.Run("apply-demo", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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

		require.NoError(k.WaitForDeployment(ctx, *namespace, "vote-bot"))
		require.NoError(k.WaitForDeployment(ctx, *namespace, "voting"))
		require.NoError(k.WaitForDeployment(ctx, *namespace, "emoji"))
		require.NoError(k.WaitForDeployment(ctx, *namespace, "web"))
	}), "applying the demo is required for subsequent tests to run")

	t.Run("test-demo", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		emojiwebIP, err := k.WaitForLoadBalancer(ctx, *namespace, "web-svc")
		require.NoError(err)

		cfg := &tls.Config{RootCAs: x509.NewCertPool()}
		pem, err := os.ReadFile(path.Join(dir, "verify", "mesh-ca.pem"))
		require.NoError(err)
		require.True(cfg.RootCAs.AppendCertsFromPEM(pem))

		c := http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(emojiwebIP, "443"))
				},
				TLSClientConfig: cfg,
			},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://web", nil)
		require.NoError(err)
		resp, err := c.Do(req)
		require.NoError(err)
		require.Equal(http.StatusOK, resp.StatusCode)
	})
}

type contrast struct {
	dir string
}

func (c *contrast) Run(t *testing.T, ctx context.Context, timeout time.Duration, args ...string) {
	require.True(t, t.Run(args[0], func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		args = append([]string{"--log-level", "debug"}, args...)
		cmd := exec.CommandContext(ctx, "./contrast", args...)
		cmd.Dir = c.dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "output:\n%s", string(out))
	}), args[0]+" needs to succeed for subsequent tests to run")
}

func randomNamespace(t *testing.T) string {
	buf := make([]byte, 4)
	n, err := rand.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	return "releasetest-" + hex.EncodeToString(buf)
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
