// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package policy

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kubeapi"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	opensslBackend  = "openssl-backend"
	opensslFrontend = "openssl-frontend"
	coordinator     = "coordinator"
	// Persistent pod identifier of StatefulSet Coordinator is used.
	coordinatorPod = "coordinator-0"
)

var (
	imageReplacementsFile, namespaceFile, platformStr string
	skipUndeploy                                      bool
)

func TestPolicy(t *testing.T) {
	platform, err := platforms.FromString(platformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, platform, skipUndeploy)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.OpenSSL()
	coordinatorBundle := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinatorBundle...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	// Apply deployment using default policies
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	t.Run("check containers without policy annotation do not start", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		t.Log("Waiting to ensure container start up failed")

		err := c.WaitForEvent(ctx, kubeclient.StartingBlocked, kubeclient.Pod{}, ct.Namespace, coordinatorPod)
		require.NoError(err)

		t.Log("Restarting container")

		require.NoError(c.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, coordinator))
		t.Log("Waiting to ensure container start up failed")

		errRst := c.WaitForEvent(ctx, kubeclient.StartingBlocked, kubeclient.Pod{}, ct.Namespace, coordinatorPod)
		require.NoError(errRst)
	})

	// initial deployment with pod allowed

	ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(1*time.Minute))
	defer cancel()

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.NoError(t, ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, coordinator))
	require.NoError(t, ct.Kubeclient.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
	require.NoError(t, ct.Kubeclient.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
	// Set always waits for the coordinator to be ready, therefore we don not require an explicit waitFor() here
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("pod cannot join after it was removed from the manifest", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), ct.FactorPlatformTimeout(1*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		t.Log("Waiting for deployments")
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

		// get the attestation failures before removing a policy
		initialFailures := getFailures(ctx, t, ct)

		t.Log("Initial failures:", initialFailures)

		// parse the manifest
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err)
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))

		// Remove a policy from the manifest.
		for policyHash := range m.Policies {
			if slices.Contains(m.Policies[policyHash].SANs, opensslFrontend) {
				delete(m.Policies, policyHash)
			}
		}

		// write the new manifest
		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "manifest.json"), manifestBytes, 0o644))

		// parse the original resources
		resourceBytes, err := os.ReadFile(path.Join(ct.WorkDir, "resources.yml"))
		require.NoError(err)
		r, err := kubeapi.UnmarshalUnstructuredK8SResource(resourceBytes)
		require.NoError(err)

		// remove everything from the openssl-frontend
		newResources := make([]*unstructured.Unstructured, 0, len(r))
		for _, resource := range r {
			name := resource.GetName()
			require.NoError(err)
			if strings.Contains(name, opensslFrontend) {
				continue
			}
			newResources = append(newResources, resource)
		}

		// write the new resources yaml
		resourceBytes, err = kuberesource.EncodeUnstructured(newResources)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yml"), resourceBytes, 0o644))

		// set the new manifest
		ct.Set(t)

		// restart the deployments
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend)) // not waiting since it would fail
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Ready, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// wait for the init container of the openssl-frontend pod to enter the running state
		require.NoError(c.WaitFor(ctx, kubeclient.InitContainersRunning, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
		newFailures := getFailures(ctx, t, ct)
		t.Log("New failures:", newFailures)
		// errors should happen
		require.Greater(newFailures, initialFailures)
	})

	t.Run("cli does not verify coordinator with unexpected policy hash", func(t *testing.T) {
		require := require.New(t)

		// read expected coordinator policy hash
		policyHashBytes, err := os.ReadFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"))
		require.NoError(err)
		require.NotEmpty(policyHashBytes)
		policyHash := make([]byte, len(policyHashBytes)/2)
		_, err = hex.Decode(policyHash, policyHashBytes)
		require.NoError(err)

		// change expected coordinator policy hash
		policyHash[0] ^= 1
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"), []byte(hex.EncodeToString(policyHash)), 0o644))

		// verification should fail
		require.ErrorContains(ct.RunVerify(), "validating report")

		// restore correct coordinator policy hash
		policyHash[0] ^= 1
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "coordinator-policy.sha256"), []byte(hex.EncodeToString(policyHash)), 0o644))
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.StringVar(&platformStr, "platform", "", "Deployment platform")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}

func getFailures(ctx context.Context, t *testing.T, ct *contrasttest.ContrastTest) int {
	require := require.New(t)

	coordPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", coordinator)
	require.NoError(err)
	require.NotEmpty(coordPods, "pod not found: %s/%s", ct.Namespace, coordinator)
	coordIP := coordPods[0].Status.PodIP
	backendPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslBackend)
	require.NoError(err)
	require.NotEmpty(backendPods, "pod not found: %s/%s", ct.Namespace, opensslBackend)
	metricsString, _, err := ct.Kubeclient.Exec(ctx, ct.Namespace, backendPods[0].Name, []string{"curl", coordIP + ":9102/metrics"})
	require.NoError(err)

	// parse the logs
	metrics, err := (&expfmt.TextParser{}).TextToMetricFamilies(strings.NewReader(metricsString))
	require.NoError(err)
	const metricName = "contrast_grpc_server_handled_total"
	metricFamily, ok := metrics[metricName]
	require.True(ok, "metric family %q not found", metricName)
	failures := 0
	for _, metric := range metricFamily.GetMetric() {
		for _, labelPair := range metric.GetLabel() {
			if labelPair.Name == nil || *labelPair.Name != "grpc_code" {
				continue
			}
			if labelPair.Value == nil || *labelPair.Value != "PermissionDenied" {
				break
			}
			failures += int(metric.GetCounter().GetValue())
		}
	}
	return failures
}
