# Azure nested CVM-in-VM internals

The AKS CoCo preview uses a publicly accessible VM image for the Kubernetes nodes: `/CommunityGalleries/AKSCBLMariner-0cf98c92-cbb1-40e7-a71c-d127332549ee/Images/V2katagen2/Versions/latest`.
There is also `/CommunityGalleries/AKSAzureLinux-f7c7cda5-1c9a-4bdc-a222-9614c968580b/Images/V2katagen2`, which is similar but not using CBL Mariner as a base.
The instance types used are from the [DCas_cc_v5 and DCads_cc_v5-series](https://learn.microsoft.com/en-us/azure/virtual-machines/dcasccv5-dcadsccv5-series) of VMs (for example `Standard_DC4as_cc_v5`).

## Using nested CVM-in-VM outside of AKS

The instance types and images can be used for regular VMs (and scale sets):

```sh
LOCATION=westeurope
RG=change-me
az vm create \
    --size Standard_DC4as_cc_v5 \
    -l "${LOCATION}" \
    --resource-group "${RG}" \
    --name nested-virt-test \
    --image /CommunityGalleries/AKSCBLMariner-0cf98c92-cbb1-40e7-a71c-d127332549ee/Images/V2katagen2/Versions/latest \
    --boot-diagnostics-storage ""
az ssh vm --resource-group "${RG}" --vm-name nested-virt-test
```

The VM has access to hyperv as a paravisor via `/dev/mshv` (and *not* `/dev/kvm`, which would be expected for nested virtualization on Linux). The user space component for spawning nested VMs on `mshv` is part of [rust-vmm](https://github.com/rust-vmm/mshv).
This feature is enabled by kernel patches which aren't yet upstream. At the time of writing, the VM image uses a kernel with a `uname` of `5.15.126.mshv9-2.cm2` and this additional kernel cmdline parameter: `hyperv_resvd_new=0x1000!0x933000,0x17400000!0x104e00000,0x1000!0x1000,0x4e00000!0x100000000`.

The Kernel is built from a CBL Mariner [spec file](https://github.com/microsoft/CBL-Mariner/blob/2.0/SPECS/kernel-mshv/kernel-mshv.spec) and the Kernel source is [publicly accessible](https://cblmarinerstorage.blob.core.windows.net/sources/core/kernel-mshv-5.15.126.mshv9.tar.gz).
MicroVM guests use a newer Kernel ([spec](https://github.com/microsoft/CBL-Mariner/tree/2.0/SPECS/kernel-uvm-cvm)).

The image also ships parts of the confidential-containers tools, the kata-runtime, cloud-hypervisor and related tools (kata-runtime, cloud-hypervisor, [containerd](https://github.com/microsoft/confidential-containers-containerd)), some of which contain patches that aren't yet upstream, as well as a custom guest image under `/opt`:

<details>
<summary><code>$ find /opt/confidential-containers/</code></summary>

```shell-session
/opt/confidential-containers/
/opt/confidential-containers/libexec
/opt/confidential-containers/libexec/virtiofsd
/opt/confidential-containers/bin
/opt/confidential-containers/bin/kata-runtime
/opt/confidential-containers/bin/kata-monitor
/opt/confidential-containers/bin/cloud-hypervisor
/opt/confidential-containers/bin/cloud-hypervisor-snp
/opt/confidential-containers/bin/kata-collect-data.sh
/opt/confidential-containers/share
/opt/confidential-containers/share/kata-containers
/opt/confidential-containers/share/kata-containers/vmlinux.container
/opt/confidential-containers/share/kata-containers/kata-containers-igvm-debug.img
/opt/confidential-containers/share/kata-containers/kata-containers.img
/opt/confidential-containers/share/kata-containers/reference-info-base64
/opt/confidential-containers/share/kata-containers/kata-containers-igvm.img
/opt/confidential-containers/share/defaults
/opt/confidential-containers/share/defaults/kata-containers
/opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml
/opt/confidential-containers/share/defaults/kata-containers/configuration-clh.toml
```
</details>

With those components, it's possible to use the `containerd-shim-kata-cc-v2` runtime for containerd (or add it as a runtime class in k8s).

<details>
<summary><code>/etc/containerd/config.toml</code></summary>

```toml
version = 2
oom_score = 0
[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "mcr.microsoft.com/oss/kubernetes/pause:3.6"
  [plugins."io.containerd.grpc.v1.cri".containerd]
      disable_snapshot_annotations = false
    default_runtime_name = "runc"
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
      runtime_type = "io.containerd.runc.v2"
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
      BinaryName = "/usr/bin/runc"
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.untrusted]
      runtime_type = "io.containerd.runc.v2"
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.untrusted.options]
      BinaryName = "/usr/bin/runc"
  [plugins."io.containerd.grpc.v1.cri".cni]
    bin_dir = "/opt/cni/bin"
    conf_dir = "/etc/cni/net.d"
    conf_template = "/etc/containerd/kubenet_template.conf"
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
  [plugins."io.containerd.grpc.v1.cri".registry.headers]
    X-Meta-Source-Client = ["azure/aks"]
[metrics]
  address = "0.0.0.0:10257"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
  runtime_type = "io.containerd.kata.v2"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.katacli]
  runtime_type = "io.containerd.runc.v1"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.katacli.options]
  NoPivotRoot = false
  NoNewKeyring = false
  ShimCgroup = ""
  IoUid = 0
  IoGid = 0
  BinaryName = "/usr/bin/kata-runtime"
  Root = ""
  CriuPath = ""
  SystemdCgroup = false
[proxy_plugins]
  [proxy_plugins.tardev]
    type = "snapshot"
    address = "/run/containerd/tardev-snapshotter.sock"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-cc]
  snapshotter = "tardev"
  runtime_type = "io.containerd.kata-cc.v2"
  privileged_without_host_devices = true
  pod_annotations = ["io.katacontainers.*"]
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-cc.options]
    ConfigPath = "/opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml"
```
</details>

<details>
<summary>Runtime options extracted by <code>crictl inspect CONTAINER_ID | jq .info</code> (truncated)</summary>

```json
{
    "snapshotKey": "070f6f2f0ec920cc9e8c050bf08730c79d4af43c640b8cfc16002b8f1e009767",
    "snapshotter": "tardev",
    "runtimeType": "io.containerd.kata-cc.v2",
    "runtimeOptions": {
      "config_path": "/opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml"
    }
}
```
</details>

With the containerd configuration from above, the following commands can be used to start the [tardev-snapshotter](https://github.com/kata-containers/tardev-snapshotter) and spawn a container using the `kata-cc` runtime class.
Please note that while this example almost works, it doesn't currently result in a working container outside of AKS.
This is probably due to a mismatch of exact arguments or configuration files.
Maybe a policy annotation is required for booting.

```
systemctl enable --now tardev-snapshotter.service
ctr image pull docker.io/library/ubuntu:latest
ctr run \
    --rm \
    --runtime "io.containerd.kata-cc.v2" \
    --runtime-config-path /opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml \
    --snapshotter tardev docker.io/library/ubuntu:latest \
    foo
```

## Kata-Containers Configuration on AKS

In the last section we learned that the `kata-cc` runtime is configured with
`/opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml`.
This file is a serialization of [katautils.tomlConfig] and contains decent inline documentation
(see [snapshot]). In case we would like to tweak configuration, [drop-in fragments] are a handy
option.

<details>
<summary>Example of a drop-in fragment raising the default memory per VM</summary>

```sh
cat >/opt/confidential-containers/share/defaults/kata-containers/config.d/10-memory.toml <<EOF
[hypervisor.clh]
default_memory = 512
EOF
```
</details>

[katautils.tomlConfig]: https://github.com/kata-containers/kata-containers/blob/40d9a65/src/runtime/pkg/katautils/config.go#L64
[snapshot]: https://gist.github.com/burgerdev/6133d56deb1722a746df2269f4d33160
[drop-in fragments]: https://github.com/kata-containers/kata-containers/blob/main/src/runtime/README.md#drop-in-configuration-file-fragments

### Resource Management

There's [AKS documentation for resource management] which explains the basics of how CPU and
memory are allocated for a Kata VM.
The default memory overhead added by the `RuntimeClass` is quite high at 2GiB, which fills up the node fast.
It's unclear why this default is chosen, given that the container limit is added on top of this
value and that the VMs are created with a 256MiB overhead.

Forcing a size with the pod annotation
`io.katacontainers.config.hypervisor.default_memory` would be possible, but the annotation would
need to be allow-listed in the config setting `enable_annotations`.

[AKS documentation for resource management]: https://learn.microsoft.com/en-us/azure/aks/confidential-containers-overview#resource-allocation-overview

<details>
<summary>Relevant config snippet</summary>

```toml
[hypervisor.clh]
default_memory = 256
default_vcpus = 1
enable_annotations = ["enable_iommu"]

[runtime]
static_sandbox_resource_mgmt = true
```
</details>

### Cloud Hypervisor

We can learn some interesting facts about the VMs managed with Cloud Hypervisor by talking to their
api sockets.

<details>
<summary>List some facts about all CH VMs</summary>

```sh
find /run/vc/vm -name clh-api.sock -exec curl -sS --unix-socket "{}" http://./api/v1/vm.info ";" |
  jq -s 'map( {
    "sock": .config.vsock.socket,
    "policy": .config.payload.host_data,
    "vcpus": .config.cpus.max_vcpus,
    "memory_mib": (.config.memory.size / 1024 / 1024),
    "disks": [.config.disks[] | .path]
    }) | sort_by(.sock)'
```

```json
[
  {
    "sock": "/run/vc/vm/c9ebe6792862527ebbfea9aa353258e0d301fffdde784bb6dde65090ddc18704/clh.sock",
    "policy": "ee6d1cb44554a792665264c9fe412d270feeff32fbc2ef40158a1f969a45cec1",
    "vcpus": 1,
    "memory_mib": 2148,
    "disks": [
      "/opt/confidential-containers/share/kata-containers/kata-containers.img",
      "/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d",
      "/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/35ff57f24506a885c071fe2aaa526aa172b051477016e1d03feccb2bbc58b8b7"
    ]
  },
  "..."
]
```
</details>

The API is documented [here](https://github.com/cloud-hypervisor/cloud-hypervisor/blob/v43.0/docs/api.md).
