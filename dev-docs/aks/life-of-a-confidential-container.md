# Life of a Confidential Container

## Example Container Image

We start with simple image comprising a `busybox` binary and two layers:
`docker.io/burgerdev/example-container:1`. A useful tool for working with images
is
[`crane`](https://github.com/google/go-containerregistry/tree/main/cmd/crane).

<details>
<summary>Image Manifest</summary>

```sh
crane manifest docker.io/burgerdev/example-container:1@sha256:cc3eca5ffd66d6d7e71db5cfd9754cd7e9a9b16627bfd285fd2f1465fb113cbe
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
  "config": {
    "mediaType": "application/vnd.docker.container.image.v1+json",
    "size": 820,
    "digest": "sha256:5416e5e66b25bac17cdf3fda3eee22a38dbd389d5a179751c92f783cc3494f90"
  },
  "layers": [
    {
      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
      "size": 766762,
      "digest": "sha256:d30d3fe99ab4055a5ad13906b94b7bda07efb5ff057b93c37d6070b54f8ed408"
    },
    {
      "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
      "size": 95,
      "digest": "sha256:d75d7b35464283bbb95710d85de7636f7cc9e69341cc541924957c7d9c2663ea"
    }
  ]
}
```

</details>

<details>
<summary>Image Configuration</summary>

Fetch the image configuration by digest (see the manifest) and print it.

```sh
crane blob docker.io/burgerdev/example-container@sha256:5416e5e66b25bac17cdf3fda3eee22a38dbd389d5a179751c92f783cc3494f90 | \
  jq '{ config: .config, rootfs: .rootfs}'
```

```json
{
  "config": {
    "Env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
    ],
    "Entrypoint": [
      "/bin/busybox"
    ],
    "WorkingDir": "/"
  },
  "rootfs": {
    "type": "layers",
    "diff_ids": [
      "sha256:d8667d954dc18da4caeb1f88f41aeada0f3856af6de9236963a2c405800b1e15",
      "sha256:ed328a6cc2d48369c6c566dd144f55455c40934abe2a7ec4395dd08af7402df7"
    ]
  }
}
```

</details>

<details>
<summary>Image Content</summary>

Fetch the layer blobs by digest (see the manifest), unpack them and print their
content.

```console
$ crane blob docker.io/burgerdev/example-container@sha256:d30d3fe99ab4055a5ad13906b94b7bda07efb5ff057b93c37d6070b54f8ed408 | tar tz
LAYER_1
bin/
bin/busybox
$ crane blob docker.io/burgerdev/example-container@sha256:d75d7b35464283bbb95710d85de7636f7cc9e69341cc541924957c7d9c2663ea | tar tz
LAYER_2
```

</details>

## Pod Definition

Next, this image is referenced in a Kubernetes pod definition.

<details>
<summary>Pod Definition</summary>

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example
  namespace: default
spec:
  runtimeClassName: kata-cc-isolation
  containers:
    - command: ["/bin/busybox", "tail", "-f", "/dev/null"]
      image: "docker.io/burgerdev/example-container:1@sha256:cc3eca5ffd66d6d7e71db5cfd9754cd7e9a9b16627bfd285fd2f1465fb113cbe"
      imagePullPolicy: Always
      name: example
      resources:
        limits:
          cpu: "0.2"
          memory: 50Mi
```

</details>

## genpolicy

We use the [`genpolicy`] tool to generate a runtime policy for the pod. This
policy defines what OCI runtime parameters are acceptable for this pod. Runtime
policies are covered in more detail in a
[dedicated document](../coco/policy.md). It also calculates a `dm-verity` hash,
but more on that later.

[`genpolicy`]: https://github.com/microsoft/kata-containers/tree/3.2.0.azl1.genpolicy0/src/tools/genpolicy

```sh
genpolicy -u -y pod.yml
```

The `-u` flag allocates a layer cache file that maps diff ids to `dm-verity`
hashes. The diff ids correspond to those in the _Image Configuration_ above.

<details>
<summary>Layer Cache</summary>

```json
[
  {
    "diff_id": "sha256:d8667d954dc18da4caeb1f88f41aeada0f3856af6de9236963a2c405800b1e15",
    "verity_hash": "a209e62eb6cfaf229cc12825f63009459d9621951f507980337ac05c68c89138"
  },
  {
    "diff_id": "sha256:ed328a6cc2d48369c6c566dd144f55455c40934abe2a7ec4395dd08af7402df7",
    "verity_hash": "3e180656327e86fa7aa220ff278695f1df2a2679e1aa80e8a454ccf0460c7d39"
  },
  {
    // pause container
    "diff_id": "sha256:9760f55e20e3f4eb6b837e1b323b3c6f29b1ef4a4617fe98625ead879e91b1c1",
    "verity_hash": "817250f1a3e336da76f5bd3fa784e1b26d959b9c131876815ba2604048b70c18"
  }
]
```

</details>

The annotated pod is applied to the Kubernetes API server, where it waits to be
scheduled.

## Kubelet

The kubelet watches pod resources and waits for a pod with `spec.nodeName`
matching its hostname. If it finds a scheduled pod for this node, the kubelet
attempts to run it. The `runtimeClassName` field specifies the `RuntimeClass`
resource to use for this pod, which maps the name (`kata-cc-isolation`) to a
handler (`kata-cc`).

<details>
<summary>Runtime Class</summary>

```yaml
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata-cc-isolation
handler: kata-cc
overhead:
  podFixed:
    memory: 2Gi
scheduling:
  nodeSelector:
    kubernetes.azure.com/kata-cc-isolation: "true"
```

</details>

The kubelet proceeds with calls to the [CRI] plugin, requesting the `kata-cc`
runtime. We can inspect the pods with the [`crictl`] tool. The allocated sandbox
id for our pod will be important in the next sections.

[CRI]: https://kubernetes.io/docs/concepts/architecture/cri/
[`crictl`]: https://kubernetes.io/docs/tasks/debug/debug-cluster/crictl/

<details>
<summary>Find Pod Sandbox ID</summary>

```console
$ crictl ps -o json | jq -r '
  .containers[] |
  select(.labels."io.kubernetes.pod.name" == "example" and .labels."io.kubernetes.pod.namespace" == "default") |
  .id'
6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed
$ crictl inspectp 6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed | \
  jq -r '.status.runtimeHandler'
kata-cc
```

</details>

## Containerd

The CRI plugin is provided by [containerd](https://containerd.io/). Runtimes are
registered in the containerd configuration.

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-cc]
  snapshotter = "tardev"
  runtime_type = "io.containerd.kata-cc.v2"
  privileged_without_host_devices = true
  pod_annotations = ["io.katacontainers.*"]
```

Before a container can be started inside the sandbox, containerd needs to set up
the root filesystem. This is the task of the configured snapshotter, in this
case `tardev`.

```console
$ ctr --namespace k8s.io container ls "runtime.name==io.containerd.kata-cc.v2"
CONTAINER                                                           IMAGE                                                                                                            RUNTIME
14d29d11b16243ce7b4d1b254b57430704d49ca8a8f9948561b08727285304cb    docker.io/burgerdev/example-container@sha256:cc3eca5ffd66d6d7e71db5cfd9754cd7e9a9b16627bfd285fd2f1465fb113cbe    io.containerd.kata-cc.v2
6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed    mcr.microsoft.com/oss/kubernetes/pause:3.6                                                                       io.containerd.kata-cc.v2
```

## Tardev Snapshotter

The [snapshot API] abstracts setting up the container root filesystem such that
it can be used by the runtime. For each layer in the image, it receives a
request from containerd to add this layers content to the final bundle. Examples
of snapshotters include Device Mapper, overlayfs, etc.

However, the [tardev snapshotter] doesn't assemble a final root. Its actual goal
is to set up block devices that can be added to the pod VM. For each layer
requested by containerd, the snapshotter creates a metadata file and a layer
file. The layer file is created by unzipping the container image layer,
appending an index and computing the dm-verity checksum over both. The resulting
file is stored and will be mounted by the Kata Runtime.

[snapshot API]: https://github.com/containerd/containerd/blob/v1.7.18/api/services/snapshots/v1/snapshots.proto
[tardev snapshotter]: https://github.com/microsoft/kata-containers/tree/3.2.0.azl1.genpolicy0/src/tardev-snapshotter

<details>
<summary>Tardev Snapshot Layout</summary>

The metadata files are stored in the
`/var/lib/containerd/io.containerd.snapshotter.v1.tardev/snapshots` directory.
Note how the `layer-digest` matches the layer digest in the _Image Manifest_
above.

```json
{
  "kind": "Committed",
  "name": "k8s.io/11/sha256:b7ec85bd39df687c569a301a484b47e71c45321bddfd5f41b38cbe2811ca9696",
  "parent": "k8s.io/9/sha256:d8667d954dc18da4caeb1f88f41aeada0f3856af6de9236963a2c405800b1e15",
  "labels": {
    "containerd.io/snapshot/cri.layer-digest": "sha256:d75d7b35464283bbb95710d85de7636f7cc9e69341cc541924957c7d9c2663ea",
    "containerd.io/snapshot/cri.manifest-digest": "sha256:cc3eca5ffd66d6d7e71db5cfd9754cd7e9a9b16627bfd285fd2f1465fb113cbe",
    "containerd.io/snapshot/cri.image-ref": "docker.io/burgerdev/example-container@sha256:cc3eca5ffd66d6d7e71db5cfd9754cd7e9a9b16627bfd285fd2f1465fb113cbe",
    "containerd.io/snapshot/cri.image-layers": "sha256:d75d7b35464283bbb95710d85de7636f7cc9e69341cc541924957c7d9c2663ea",
    "io.katacontainers.dm-verity.root-hash": "3e180656327e86fa7aa220ff278695f1df2a2679e1aa80e8a454ccf0460c7d39",
    "containerd.io/snapshot.ref": "sha256:b7ec85bd39df687c569a301a484b47e71c45321bddfd5f41b38cbe2811ca9696"
  },
  "created_at": {
    "secs_since_epoch": 1719419039,
    "nanos_since_epoch": 704129318
  },
  "updated_at": {
    "secs_since_epoch": 1719419039,
    "nanos_since_epoch": 704129418
  }
}
```

The corresponding tarfs file can be found in
`/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers` under the same
name.

```console
$ tar tf /var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc
LAYER_2
```

</details>

## Kata Runtime

containerd starts the Kata Shim and translates the CRI methods to [Runtime v2]
methods.

<!-- TODO(burgerdev): this section is a stub. -->

[Runtime v2]: https://github.com/containerd/containerd/blob/main/core/runtime/v2/README.md

## Cloud Hypervisor

Cloud Hypervisor manages the pod VMs in directories under `/run/vc/vm/`. The pod
sandbox id from earlier can be used to query the [REST API] and inspect VM
configuration. Alternatively, the [`ch-remote`] binary should support most
operations, too.

[REST API]: https://github.com/cloud-hypervisor/cloud-hypervisor/blob/v40.0/docs/api.md#external-api
[`ch-remote`]: https://github.com/cloud-hypervisor/cloud-hypervisor/blob/v40.0/src/bin/ch-remote.rs

<details>
<summary>VM Info</summary>

We query the API endpoint using the sandbox id obtained by `crictl`. Among lots
of other details, we learn that there are 4 "disks" mounted:

- The root image.
- The indexed tarball for the pause container.
- Two indexed tarballs for the main container, one for each image layer.

```sh
curl -s --unix-socket /run/vc/vm/6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed/clh-api.sock -X GET http://localhost/api/v1/vm.info | \
  jq '[ .config.disks[] | { path: .path, id: .id } ]'
```

```json
[
  {
    "path": "/opt/confidential-containers/share/kata-containers/kata-containers.img",
    "id": "_disk0"
  },
  {
    "path": "/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d",
    "id": "_disk3"
  },
  {
    "path": "/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc",
    "id": "_disk4"
  },
  {
    "path": "/var/lib/containerd/io.containerd.snapshotter.v1.tardev/layers/20fa1959f77bb8fd725123f59d63373051c833e1d3f3e3ac51be0169c71f9b9c",
    "id": "_disk5"
  }
]
```

</details>

## Kata Agent

Inside the confidential VM, the Kata Agent starts up and serves the [agent API].
If we configured Kata for [debug mode](../serial-console.md), we can log into
the container by sandbox id:

```sh
kata-runtime exec 6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed
```

[agent API]: https://github.com/microsoft/kata-containers/blob/3.2.0.azl1.genpolicy0/src/libs/protocols/protos/agent.proto

<details>
<summary>Block Devices in Guest</summary>

Taking a look around, we see that the block devices are present and mapped with
dm-verity. Note that the dm-verity hash matches both tardev snapshot metadata
and genpolicy layer metadata.

```console
$ dmsetup ls
20fa1959f77bb8fd725123f59d63373051c833e1d3f3e3ac51be0169c71f9b9c        (253:3)
2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc        (253:2)
5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d        (253:1)
dm-verity       (253:0)
$ ls -l /dev/ | grep 253
brw-rw---- 1 root disk    253,   0 Jun 26 16:25 dm-0
brw-rw---- 1 root disk    253,   1 Jun 26 16:25 dm-1
brw-rw---- 1 root disk    253,   2 Jun 26 16:25 dm-2
brw-rw---- 1 root disk    253,   3 Jun 26 16:25 dm-3
$ tar tf /dev/dm-2
LAYER_2
$ dmsetup measure 2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc
0 16 verity target_name=verity,target_version=1.9.0,hash_failed=V,verity_version=1,data_device_name=254:32,hash_device_name=254:32,verity_algorithm=sha256,root_digest=3e180656327e86fa7aa220ff278695f1df2a2679e1aa80e8a454ccf0460c7d39,salt=0000000000000000000000000000000000000000000000000000000000000000,ignore_zero_blocks=n,check_at_most_once=n;
```

</details>

## Tarfs

The integrity protected tarball is directly mounted as a filesystem. The
filesystem driver belongs to the [`tarfs`] module and is loaded into the guest
kernel.

[`tarfs`]: https://github.com/microsoft/kata-containers/tree/3.2.0.azl1.genpolicy0/src/tarfs

<details>
<summary>Mount Points</summary>

```console
$ lsmod
Module                  Size  Used by
tarfs                  16384  -2
$ cat /proc/filesystems | grep tar
        tar
$ mount | grep -F sandbox/layers
/dev/mapper/5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d on /run/kata-containers/sandbox/layers/5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d type tar (ro,relatime)
/dev/mapper/2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc on /run/kata-containers/sandbox/layers/2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc type tar (ro,relatime)
/dev/mapper/20fa1959f77bb8fd725123f59d63373051c833e1d3f3e3ac51be0169c71f9b9c on /run/kata-containers/sandbox/layers/20fa1959f77bb8fd725123f59d63373051c833e1d3f3e3ac51be0169c71f9b9c type tar (ro,relatime)
```

</details>

## Container

Before the container can finally start, the individual layers need to be
assembled into a root filesystem. This is done by the Kata agent, with a
strategy much like the `overlayfs` snapshotter.

<details>
<summary>Container Root Filesystems</summary>

```console
$ mount | grep rootfs
none on /run/kata-containers/6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed/rootfs type overlay (rw,relatime,lowerdir=5a5aad80055ff20012a50dc25f8df7a29924474324d65f7d5306ee8ee27ff71d,upperdir=/run/kata-containers/6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed/upper,workdir=/run/kata-containers/6aa04b0c9483817bb9da7216c352348d33d6fda1ecd10ac5c836bcd4a1652bed/work)
none on /run/kata-containers/14d29d11b16243ce7b4d1b254b57430704d49ca8a8f9948561b08727285304cb/rootfs type overlay (rw,relatime,lowerdir=2ead9678c1b2b8595710d3470068107cb66cf94006c8ea926da38860d26ac6bc:20fa1959f77bb8fd725123f59d63373051c833e1d3f3e3ac51be0169c71f9b9c,upperdir=/run/kata-containers/14d29d11b16243ce7b4d1b254b57430704d49ca8a8f9948561b08727285304cb/upper,workdir=/run/kata-containers/14d29d11b16243ce7b4d1b254b57430704d49ca8a8f9948561b08727285304cb/work,index=off,xino=off)
$ ls -l /run/kata-containers/14d29d11b16243ce7b4d1b254b57430704d49ca8a8f9948561b08727285304cb/rootfs
-rw-r--r-- 1 root root   0 Jun 26 13:53 LAYER_1
-rw-r--r-- 1 root root   0 Jun 26 13:53 LAYER_2
drwxr-xr-x 1 root root  32 Jun 26 13:55 bin
drwxr-xr-x 2 root root  40 Jun 26 16:25 dev
drwxr-xr-x 2 root root 100 Jun 26 16:25 etc
drwxr-xr-x 2 root root  40 Jun 26 16:25 proc
drwxr-xr-x 2 root root  40 Jun 26 16:25 sys
drwxr-xr-x 3 root root  60 Jun 26 16:25 var
```

</details>
