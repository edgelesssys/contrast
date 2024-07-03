// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

///go:build e2e

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

		time.Sleep(5 * time.Second) // let the error counter go up initially

		// get the attestation failures before removing a policy
		initialFailures := getFailures(t, ctx, ct)

		t.Log("Initial failures:", initialFailures)

		// parse the manifest
		manifestBytes, err := os.ReadFile(path.Join(ct.WorkDir, "manifest.json"))
		require.NoError(err)
		var m manifest.Manifest
		require.NoError(json.Unmarshal(manifestBytes, &m))

		// Remove a policy from the manifest.
		newPolicies := make(map[manifest.HexString][]string)
		for policyHash := range m.Policies {
			if slices.Contains(m.Policies[policyHash], opensslFrontend) {
				continue
			}
			newPolicies[policyHash] = m.Policies[policyHash]
		}
		m.Policies = newPolicies

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
		newResources := r[:0]
		for _, resource := range r {
			name := resource.GetName()
			require.NoError(err)
			if name == opensslFrontend || name == "port-forwarder-"+opensslFrontend {
				continue
			}
			newResources = append(newResources, resource)
		}

		// write the new resources yaml
		r = newResources
		resourceBytes, err = kuberesource.EncodeUnstructured(r)
		require.NoError(err)
		require.NoError(os.WriteFile(path.Join(ct.WorkDir, "resources.yaml"), resourceBytes, 0o644))

		// set the new manifest
		ct.Set(t)

		// restart the deployments
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslFrontend)) // not waiting since it would fail
		require.NoError(c.Restart(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))
		require.NoError(c.WaitFor(ctx, kubeclient.Deployment{}, ct.Namespace, opensslBackend))

		// wait a bit to let the attestation failure counter go up
		time.Sleep(5 * time.Second)

		newFailures := getFailures(t, ctx, ct)
		t.Log("New failures:", newFailures)
		// errors should happen
		require.True(newFailures > initialFailures)
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

func getFailures(t *testing.T, ctx context.Context, ct *contrasttest.ContrastTest) int {
	require := require.New(t)

	coordPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", coordinator)
	require.NotEmpty(coordPods, "pod not found: %s/%s", ct.Namespace, coordinator)
	coordIp := coordPods[0].Status.PodIP
	backendPods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, opensslBackend)
	require.NotEmpty(backendPods, "pod not found: %s/%s", ct.Namespace, opensslBackend)
	metricsString, _, err := ct.Kubeclient.Exec(ctx, ct.Namespace, backendPods[0].Name, []string{"curl", coordIp + ":9102/metrics"})
	require.NoError(err)
	metrics, err := parsePrometheus(metricsString)
	require.NoError(err)
	var failures int
	for k, v := range metrics {
		if k == "contrast_meshapi_attestation_failures" {
			failures = int(v.GetMetric()[0].GetCounter().GetValue())
		}
	}
	return failures
}
