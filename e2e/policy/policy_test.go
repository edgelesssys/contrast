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
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const opensslBackend = "openssl-backend"
const opensslFrontend = "openssl-frontend"
const coordinator = "coordinator"

var (
	imageReplacementsFile, namespaceFile string
	skipUndeploy                         bool
)

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	resources := kuberesource.OpenSSL()

	coordinatorBundle := kuberesource.CoordinatorBundle()
	resources = append(resources, coordinatorBundle...)
	resources = kuberesource.AddPortForwarders(resources)
	// resources = kuberesource.PatchCoordinatorMetrics(resources, 8080)

	// TODO: Set all resources, wait for them, get counter from init container, remove policy, restart pod, get counter again and compare
	// counter should go up

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

		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend))
		// require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, coordinator))

		// get the attestation failures before removing a policy
		coordPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", coordinator)
		require.NotEmpty(coordPods, "pod not found: %s/%s", ct.Namespace, coordinator)
		coordIp := coordPods[0].Status.PodIP
		t.Log("coordinator IP:", coordIp)
		backendPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslBackend)
		require.NotEmpty(backendPods, "pod not found: %s/%s", ct.Namespace, opensslBackend)
		metricsString, _, err := c.Exec(ctx, ct.Namespace, backendPods[0].Name, []string{"curl", coordIp + ":9102/metrics"})
		require.NoError(err)
		metrics, err := parsePrometheus(metricsString)
		require.NoError(err)
		var initialAttestationFailures int
		for k, v := range metrics {
			if k == "contrast_meshapi_attestation_failures" {
				t.Log("metric:", v.GetMetric()[0])
				initialAttestationFailures = int(v.GetMetric()[0].GetCounter().GetValue())
			}
		}
		t.Log("Initial failures:", initialAttestationFailures)

		// parse the manifest
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err)
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))
		t.Log("original manifest:", string(manifestBytes))

		// Remove a policy from the manifest.
		newPolicies := make(map[manifest.HexString][]string)
		for policyHash := range m.Policies {
			if slices.Contains(m.Policies[policyHash], opensslBackend) {
				continue
			}
			newPolicies[policyHash] = m.Policies[policyHash]
		}
		m.Policies = newPolicies

		// write the new manifest
		manifestBytes, err = json.Marshal(m)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "manifest.json"), manifestBytes, 0o644))
		t.Log("new manifest:", string(manifestBytes))

		resourceBytes, err := os.ReadFile(path.Join(ct.WorkDir, "resources.yaml"))
		require.NoError(err)
		r, err := kubeapi.UnmarshalUnstructuredK8SResource(resourceBytes)
		require.NoError(err)
		for i, resource := range r {
			name, found, err := unstructured.NestedString(resource.Object, "metadata", "name")
			require.NoError(err)
			t.Log("iterating ", name)
			if found {
				if name == opensslFrontend || name == "port-forwarder-"+opensslFrontend {
					r = slices.Delete(r, i, i)
				}
			}
		}
		resourceBytes, err = kuberesource.EncodeUnstructured(r)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yaml"), resourceBytes, 0o644))

		// set the new manifest
		ct.Set(t)

		// restart a deployment - this should fail since the manifest disallows the hash
		require.Error(c.RestartDeployment(ctx, ct.Namespace, opensslFrontend))
	})
}

func TestMain(m *testing.M) {
	flag.StringVar(&imageReplacementsFile, "image-replacements", "", "path to image replacements file")
	flag.StringVar(&namespaceFile, "namespace-file", "", "file to store the namespace in")
	flag.BoolVar(&skipUndeploy, "skip-undeploy", false, "skip undeploy step in the test")
	flag.Parse()

	os.Exit(m.Run())
}

func parsePrometheus(input string) (map[string]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(strings.NewReader(input))
}
