// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchContainerdConfig(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	tmpDir, err := os.MkdirTemp("", "patch-containerd-config-test")
	require.NoError(err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	configPath := filepath.Join(tmpDir, "config.toml")

	require.NoError(patchContainerdConfig("my-runtime", "/opt/edgeless/my-runtime", configPath))

	configData, err := os.ReadFile(configPath)
	require.NoError(err)
	assert.Equal(`version = 2
root = ''
state = ''
temp = ''
plugin_dir = ''
disabled_plugins = []
required_plugins = []
oom_score = 0
imports = []

[metrics]
address = '0.0.0.0:10257'

[plugins]
[plugins.'io.containerd.grpc.v1.cri']
sandbox_image = 'mcr.microsoft.com/oss/kubernetes/pause:3.6'

[plugins.'io.containerd.grpc.v1.cri'.cni]
bin_dir = '/opt/cni/bin'
conf_dir = '/etc/cni/net.d'
conf_template = '/etc/containerd/kubenet_template.conf'

[plugins.'io.containerd.grpc.v1.cri'.containerd]
default_runtime_name = 'runc'
disable_snapshot_annotations = false

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes]
[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.kata]
runtime_type = 'io.containerd.kata.v2'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.kata-cc]
pod_annotations = ['io.katacontainers.*']
privileged_without_host_devices = true
runtime_type = 'io.containerd.kata-cc.v2'
snapshotter = 'tardev'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.kata-cc.options]
ConfigPath = '/opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.katacli]
runtime_type = 'io.containerd.runc.v1'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.katacli.options]
BinaryName = '/usr/bin/kata-runtime'
CriuPath = ''
IoGid = 0
IoUid = 0
NoNewKeyring = false
NoPivotRoot = false
Root = ''
ShimCgroup = ''
SystemdCgroup = false

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.my-runtime]
runtime_type = 'io.containerd.contrast-cc.v2'
runtime_path = '/opt/edgeless/my-runtime/bin/containerd-shim-contrast-cc-v2'
pod_annotations = ['io.katacontainers.*']
privileged_without_host_devices = true
snapshotter = 'tardev'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.my-runtime.options]
ConfigPath = '/opt/edgeless/my-runtime/etc/configuration-clh-snp.toml'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.runc]
runtime_type = 'io.containerd.runc.v2'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.runc.options]
BinaryName = '/usr/bin/runc'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.untrusted]
runtime_type = 'io.containerd.runc.v2'

[plugins.'io.containerd.grpc.v1.cri'.containerd.runtimes.untrusted.options]
BinaryName = '/usr/bin/runc'

[plugins.'io.containerd.grpc.v1.cri'.registry]
config_path = '/etc/containerd/certs.d'

[plugins.'io.containerd.grpc.v1.cri'.registry.headers]
X-Meta-Source-Client = ['azure/aks']

[proxy_plugins]
[proxy_plugins.tardev]
type = 'snapshot'
address = '/run/containerd/tardev-snapshotter.sock'
`, string(configData))
}
