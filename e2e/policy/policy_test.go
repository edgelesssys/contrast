// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package policy

import (
	"context"
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
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	opensslBackend  = "openssl-backend"
	opensslFrontend = "openssl-frontend"
	coordinator     = "coordinator"
)

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	resources := kuberesource.OpenSSL()

	coordinatorBundle := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinatorBundle...)
	//	resources = kuberesource.AddPortForwarders(resources)
	resources = kuberesource.AddAllPortsForwarder(resources, []int32{123, 456})

	ct.Init(t, resources)

	// initial deployment with pod allowed
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("pod cannot join after it was removed from the manifest", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		c := kubeclient.NewForTest(t)

		t.Log("Waiting for deployments")
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))

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
			if slices.Contains(m.Policies[policyHash], opensslFrontend) {
				delete(m.Policies, policyHash)
			}
		}

		// write the new manifest
		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "manifest.json"), manifestBytes, 0o644))

		// parse the original resources
		resourceBytes, err := os.ReadFile(path.Join(ct.WorkDir, "resources.yaml"))
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
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yaml"), resourceBytes, 0o644))

		// set the new manifest
		ct.Set(t)

		// restart the deployments
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend)) // not waiting since it would fail
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// wait for the init container of the openssl-frontend pod to enter the running state
		ready := false
		for !ready {
			time.Sleep(1 * time.Second)
			pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslFrontend)
			require.NoError(err)
			require.NotEmpty(pods, "pod not found: %s/%s", ct.Namespace, opensslFrontend)
			require.NotEmpty(pods[0].Status.InitContainerStatuses, "pod doesn't contain init container statuses: %s/%s", ct.Namespace, opensslFrontend)
			ready = pods[0].Status.InitContainerStatuses[0].State.Running != nil
		}
		newFailures := getFailures(ctx, t, ct)
		t.Log("New failures:", newFailures)
		// errors should happen
		require.Greater(newFailures, initialFailures)
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
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
	failures := -1
	for k, v := range metrics {
		if k == "contrast_meshapi_attestation_failures_total" {
			failures = int(v.GetMetric()[0].GetCounter().GetValue())
		}
	}
	if failures == -1 {
		// metric not found
		t.Error("metric \"contrast_meshapi_attestation_failures_total\" not found")
	}
	return failures
}
