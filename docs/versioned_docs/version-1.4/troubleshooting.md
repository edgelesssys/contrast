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
    image: "ghcr.io/edgelesssys/contrast/coordinator:v1.4.1@sha256:b524bd79efb874437578e38245988ad5d1a36ba99b05bdd0fa55f73da01979b4"
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
