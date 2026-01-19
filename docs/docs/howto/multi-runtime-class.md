# Multi-runtime class deployments

This guide shows how to use more than one runtime class with a Contrast deployment.

## Applicability

For mixed SEV-SNP and TDX clusters, or on GPU-enabled clusters where not all workloads require access to a GPU.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)
2. [Install CLI](./install-cli.md)

## How-to

Depending on the configuration of your cluster and the workloads to deploy, it can be desirable to use different runtime classes for pods in the same deployment.
For example, in a cluster consisting of both SEV-SNP and TDX machines, you might want to distribute the workload over all nodes.
This guide walks you through the process of configuring your Contrast deployment accordingly.

### Deploy the required Contrast runtimes

The [runtime deployment guide](./workload-deployment/runtime-deployment.md) discussed how to deploy a single Contrast runtime, depending on the target platform and whether GPU support was required.
In preparation of a multi-runtime class deployment, we first need to apply all required runtime classes.
For example, if you are working with a cluster consisting of both SEV-SNP and TDX machines, obtain each of the two corresponding runtimes:
```sh
mkdir -p resources/runtime
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-snp.yml --output-dir resources/runtime
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-tdx.yml --output-dir resources/runtime
```

Since Kubernetes by itself has no knowledge of which runtime class belongs to which nodes, you need to manually restrict them.
Kubernetes provides [multiple mechanisms for this purpose](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/); a natural contender is to use node selectors to restrict the runtime classes to their intended platforms.
First, label your nodes, for example:
```sh
kubectl label nodes <snp_node_name> hardware=sev-snp
kubectl label nodes <tdx_node_name> hardware=tdx
```
Now restrict the runtime classes to those nodes:
```yaml
kind: RuntimeClass # node.k8s.io/v1
metadata:
  name: contrast-cc-metal-qemu-snp-<hash>
scheduling:
  nodeSelector:
    hardware: sev-snp
```
```yaml
kind: RuntimeClass # node.k8s.io/v1
metadata:
  name: contrast-cc-metal-qemu-tdx-<hash>
scheduling:
  nodeSelector:
    hardware: tdx
```
Finally, apply the runtimes:
```sh
kubectl apply -f resources/runtime/
```

This restricts the runtime classes to the nodes they support.
As a result, Kubernetes will schedule workloads using these runtime classes only on the correct nodes.

### Add the Coordinator

Following the same steps as for a single-runtime class deployment, [add the Coordinator to your resources](./workload-deployment/add-coordinator.md).

### Prepare deployment files

The steps given in the [deployment files preparation section](./workload-deployment/deployment-file-preparation.md) apply to the multi-runtime class scenario as well,
apart from [adding the `RuntimeClass`](./workload-deployment/deployment-file-preparation.md#runtimeclass), which deviates from the single-runtime case as follows.

In the single-runtime class case, the `contrast-cc` placeholder was added as the `runtimeClassName` to each pod:
```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc
```

In the multi-runtime class case, a placeholder of higher specificity is used to indicate to Contrast which runtime should be used for each pod.
For each supported platform, the corresponding `RuntimeClass` can be indicated as follows:

| Platform                             | `runtimeClassName`               |
| ------------------------------------ | -------------------------------- |
| bare metal SEV-SNP                   | `contrast-cc-metal-qemu-snp`     |
| bare metal SEV-SNP, with GPU support | `contrast-cc-metal-qemu-snp-gpu` |
| bare metal TDX                       | `contrast-cc-metal-qemu-tdx`     |
| bare metal TDX, with GPU support     | `contrast-cc-metal-qemu-tdx-gpu` |

In the mixed cluster example given above, you would set the runtime class of each of your pods to one of the following:
```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc-metal-qemu-snp
```
```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc-metal-qemu-tdx
```

:::tip

You can still use just `contrast-cc` as the `runtimeClassName`. Just as in the single-runtime case, the `RuntimeClass` will then be set depending on the `--reference-values` argument passed to `contrast generate`.

:::

:::info

From the above description, one of the limitations of multi-runtime class support becomes evident:
the runtime a pod should be used needs to be known at deployment time.

It's not possible to, for example, have a pod running on a TDX node fail over to a SEV-SNP host, even in a multi-runtime class enabled Contrast deployment.

:::

Since we restricted the runtime classes to supported nodes ([see above](#deploy-the-required-contrast-runtimes)), Kubernetes will automatically schedule the workloads to run only on compatible nodes.

Continue with the remaining steps outlined in [deployment files preparation section](./workload-deployment/deployment-file-preparation.md).

### Generating the manifest

In the [single-runtime case](./workload-deployment/generate-annotations.md), the platform is passed to `contrast generate`:
```sh
contrast generate --reference-values <platform> resources/
```
This argument has no effect on multi-runtime class deployments *if* each resource has a `runtimeClassName` listed in [the table above](#prepare-deployment-files) set.
If your deployment files contain resources with the lower-specificity `contrast-cc` placeholder, pass `--reference-values` to `contrast generate`.
The specified platform will be used to determine the runtime class only for these cases.
The argument doesn't overwrite the higher-specificity `runtimeClassNames`.

After successfully running `contrast generate` for the first time, you might be prompted to fill in the reference values for one or more of the used platforms.
This works analogously to the [single-runtime case](./workload-deployment/generate-annotations.md).
