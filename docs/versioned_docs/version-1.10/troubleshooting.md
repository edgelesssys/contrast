# Troubleshooting

This section contains information on how to debug your Contrast deployment.

## Logging

Collecting logs can be a good first step to identify problems in your
deployment. Both the CLI and the Contrast Coordinator as well as the Initializer
can be configured to emit additional logs.

### CLI

The CLI logs can be configured with the `--log-level` command-line flag, which
can be set to either `debug`, `info`, `warn` or `error`. The default is `info`.
Setting this to `debug` can get more fine-grained information as to where the
problem lies.

### Coordinator and Initializer

The logs from the Coordinator and the Initializer can be configured via the
environment variables `CONTRAST_LOG_LEVEL`, `CONTRAST_LOG_FORMAT` and
`CONTRAST_LOG_SUBSYSTEMS`.

- `CONTRAST_LOG_LEVEL` can be set to one of either `debug`, `info`, `warn`, or
  `error`, similar to the CLI (defaults to `info`).
- `CONTRAST_LOG_FORMAT` can be set to `text` or `json`, determining the output
  format (defaults to `text`).
- `CONTRAST_LOG_SUBSYSTEMS` is a comma-separated list of subsystems that should
  be enabled for logging, which are disabled by default. Subsystems include:
   `kds-getter`, `issuer` and `validator`.
  To enable all subsystems, use `*` as the value for this environment variable.
  Warnings and error messages from subsystems get printed regardless of whether
  the subsystem is listed in the `CONTRAST_LOG_SUBSYSTEMS` environment variable.

To configure debug logging with all subsystems for your Coordinator, add the
following variables to your container definition.

```yaml
spec: # v1.PodSpec
  containers:
    image: "ghcr.io/edgelesssys/contrast/coordinator:v1.10.0@sha256:a5572d6fab1d140398ce8e2247aa5aa9d12445bf4d95d02d8e6b0d8e207039c8"
    name: coordinator
    env:
    - name: CONTRAST_LOG_LEVEL
      value: debug
    - name: CONTRAST_LOG_SUBSYSTEMS
      value: "*"
    # ...
```

:::info

While the Contrast Coordinator has a policy that allows certain configurations,
the Initializer and service mesh don't. When changing environment variables of other
parts than the Coordinator, ensure to rerun `contrast generate` to update the policy.

:::

To access the logs generated by the Coordinator, you can use `kubectl` with the
following command:

```sh
kubectl logs <coordinator-pod-name>
```

## `contrast generate` returns errors

Some workload configurations are known to be insecure or incompatible with Contrast.
If such a configuration is detected during policy generation, an error is logged and the command fails.

### Images with VOLUME declarations but without a Kubernetes mount

During `contrast generate`, an error like the following is printed and the process returns with a non-zero exit code:

```
level=ERROR msg="The following volumes declared in image config don't have corresponding Kubernetes mounts: [\"/data\"]"
```

This error indicates that the container image declares a [`VOLUME`], but there is no Kubernetes volume mounted at that path (`/data` in the example).
Since it's not clearly specified if or what a container runtime is supposed to mount in that case, all declared volumes need to have a corresponding explicit Kubernetes volume mount.
Depending on the needs of the application, this could either be an [`emptyDir`] or a [Contrast-managed persistent volume].

[`VOLUME`]: https://github.com/opencontainers/image-spec/blob/06e6b47e2ef69021d9f9bf2cfa5fe43a7e010c81/config.md?plain=1#L168-L170
[`emptyDir`]: https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
[Contrast-managed persistent volume]: architecture/secrets.md#secure-persistence

## Pod fails to start

If the Coordinator or a workload pod fails to even start, it can be helpful to
look at the events of the pod during the startup process using the `describe`
command.

```sh
kubectl -n <namespace> events --for pod/<coordinator-pod-name>
```

Example output:

```
LAST SEEN  TYPE     REASON  OBJECT             MESSAGE
32m        Warning  Failed  Pod/coordinator-0  kubelet  Error: failed to create containerd task: failed to create shim task: "CreateContainerRequest is blocked by policy: ...
```

A common error, as in this example, is that the container creation was blocked by the
policy. Potential reasons are a modification of the deployment YAML without updating
the policies afterward, or a version mismatch between Contrast components.

### Regenerating the policies

To ensure there isn't a mismatch between Kubernetes resource YAML and the annotated
policies, rerun

```sh
contrast generate
```

on your deployment. If any of the policy annotations change, re-deploy with the updated policies.

### Pin container images

When generating the policies, Contrast will download the images specified in your deployment
YAML and include their cryptographic identity. If the image tag is moved to another
container image after the policy has been generated, the image downloaded at deploy time
will differ from the one at generation time, and the policy enforcement won't allow the
container to be started in the pod VM.

To ensure the correct image is always used, pin the container image to a fixed `sha256`:

```yaml
image: ubuntu:22.04@sha256:19478ce7fc2ffbce89df29fea5725a8d12e57de52eb9ea570890dc5852aac1ac
```

This way, the same image will still be pulled when the container tag (`22.04`) is moved
to another image.

### Validate Contrast components match

A version mismatch between Contrast components can cause policy validation or attestation
to fail. Each Contrast runtime is identifiable based on its (shortened) measurement value
used to name the runtime class version.

First, analyze which runtime class is currently installed in your cluster by running

```sh
kubectl get runtimeclasses
```

This should give you output similar to the following one.

```sh
NAME                                           HANDLER                                        AGE
contrast-cc-aks-clh-snp-7173acb5               contrast-cc-aks-clh-snp-7173acb5               23h
kata-cc-isolation                              kata-cc                                        45d
```

The output shows that there are four Contrast runtime classes installed (as well as the runtime class provided
by the AKS CoCo preview, which isn't used by Contrast).

Next, check if the pod that won't start has the correct runtime class configured, and the
Coordinator uses the exact same runtime:

```sh
kubectl -n <namespace> get -o=jsonpath='{.spec.runtimeClassName}' pod/<pod-name>
kubectl -n <namespace> get -o=jsonpath='{.spec.runtimeClassName}' pod/<coordinator-pod-name>
```

The output should list the runtime class the pod is using:

```sh
contrast-cc-aks-clh-snp-7173acb5
```

Version information about the currently used CLI can be obtained via the `version` flag:

```sh
contrast --version
```

```sh
contrast version v0.X.0

    runtime handler:      contrast-cc-aks-clh-snp-7173acb5
    launch digest:        beee79ca916b9e5dc59602788cbfb097721cde34943e1583a3918f21011a71c47f371f68e883f5e474a6d4053d931a35
    genpolicy version:    3.2.0.azl1.genpolicy0
    image versions:       ghcr.io/edgelesssys/contrast/coordinator@sha256:...
                          ghcr.io/edgelesssys/contrast/initializer@sha256:...
```

### Contrast attempts to pull the wrong image reference

Containerd versions before `v2.0.0` have a bug that can lead to pulling image references that differ from the PodSpec.
The policy failure contains a line starting with `allow_create_container_input` at the very top.
This is the request received from the runtime and subject to policy enforcement.
The JSON contains a list of annotations nested under `.OCI.Annotations`.
Verify that the value for annotation key `io.kubernetes.cri.image-name` corresponds to an image in your PodSpec.
If it doesn't, you need to remove that image entirely from the affected node, for example with `crictl`.

```sh
crictl rmi $IMAGE
```

Upstream backport that's fixing the bug is pending: https://github.com/containerd/containerd/pull/11644.

## VM runs out of memory

Since pod VMs are statically sized, it's easier to run out of memory due to misconfigurations.
Setting the right memory limits is even more important on bare metal, where the image layers need to be stored in the guest memory, too.
If you see an error message like this, the VM doesn't have enough space to pull images:

```
LAST SEEN   TYPE      REASON      OBJECT                            MESSAGE
2m31s       Warning   Failed      Pod/my-pod-76dc84fc75-6xn7s   Error: failed to create containerd task: failed to create shim task: failed to handle layer: hasher sha256: failed to unpack [...] No space left on device (os error 28)
```

This error can be resolved by increasing the memory limit of the containers, see the [Workload deployment](deployment.md#pod-resources) guide.

## Connection to Coordinator fails

Connections from the CLI to the Coordinator may fail due to a variety of reasons.
If the error happens during the attested TLS handshake, it will usually be reported as an error message of the following form:
`rpc error: code = <GRPC ERROR CODE> desc = connection error: desc = "<DESCRIPTION>"`.
The following table explains the reason for the error and suggests further debugging steps.

| Description  | Cause | Next steps |
| ------------ | ----- | ---------- |
| `transport: authentication handshake failed: EOF` | Connection was closed before the Coordinator could send a certificate. | Check the load balancer. |
| `received context error while waiting for new LB policy update: context deadline exceeded` | The Coordinator didn't send attestation documents before the deadline. | Check the Coordinator logs for issuer problems. |
| `transport: authentication handshake failed: remote error: tls: internal error` | Coordinator failed to issue attestation documents | Check the Coordinator logs for issuer problems. |
| `transport: authentication handshake failed: no valid attestation document certificate extensions found` | Coordinator served an unexpected certificate. | Check whether remote end is the Coordinator with port 1313; Compare versions of Coordinator and CLI. |
| `transport: authentication handshake failed: tls: first record does not look like a TLS handshake` | Coordinator didn't serve TLS. | Check whether remote end is the Coordinator with port 1313. |
| `transport: Error while dialing: dial tcp <host:port>: connect: connection refused` | Coordinator port is closed. | Check connectivity to the Coordinator; Check coordinator readiness; Check load balancer is pointing to the Coordinator port 1313. |
| `transport: authentication handshake failed: [...] validator tdx-0 failed: validating report data: quote field MR_CONFIG_ID is [...]. Expect [...]"` | Wrong Coordinator policy hash. | Compare versions of Coordinator and CLI |
