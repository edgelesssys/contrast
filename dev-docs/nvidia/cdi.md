# CDI

This page explains how devices end up in containers when using CDI annotations.
Reading [life of a confidential container](../aks/life-of-a-confidential-container.md) first is recommended to understand the flow presented here.

## The journey begins

We want to run a workload with a GPU, using CDI.
How do we start?

First, we need a pod spec.
Note the two fields marked with comments.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-tester
  annotations:
    cdi.k8s.io/gpu: nvidia.com/pgpu=0 # <== CDI annotation
spec:
  runtimeClassName: contrast-cc
  containers:
  - command: ["/bin/sh", "-c", "sleep inf"]
    image: ghcr.io/edgelesssys/contrast/ubuntu:24.04@sha256:0f9e2b7901aa01cf394f9e1af69387e2fd4ee256fd8a95fb9ce3ae87375a31e6
    name: main
    resources:
      limits:
        nvidia.com/GH100_H100_PCIE: "1" # <== extended resource
```

Although these two fields look very similar, they serve related but orthogonal purposes, explained in the next subsections.

### Extended resources

Kubernetes has a notion of [extended resources].
These resources are an abstract concept for the Kubernetes scheduler, which is only responsible for the bookkeeping.
How the pod actually obtains these resources is out of the extended resources' scope.

Extended resources are registered as _node capacities_, either by a [device plugin] or manually by patching the node object.

```sh
kubectl get nodes discovery -o json | jq .status.capacity
```

```json
{
  "cpu": "32",
  "ephemeral-storage": "959218776Ki",
  "feature.node.kubernetes.io/sev_asids": "907",
  "feature.node.kubernetes.io/sev_es": "99",
  "hugepages-1Gi": "0",
  "hugepages-2Mi": "0",
  "memory": "65154724Ki",
  "nvidia.com/GH100_H100_PCIE": "1",
  "pods": "110"
}
```

The `sev_asids` in the status above are a good example for a resource that should be _accounted for_, but doesn't need to be _allocated by_ Kubernetes, because it's implicitly consumed by starting a Kata CVM.
Similarly, the GPU is published and nominally consumed by an extended resource, but it's actually allocated by the CDI handler.

[extended resources]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#extended-resources
[device plugin]: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins

### Container device interface

[CDI] is a specification for automatic modifications of containers based on:

1. A device request, in the form of a magic annotation.
2. The specification, consisting of a list of available devices and the necessary container modifications to use them.

[CDI]: https://github.com/cncf-tags/container-device-interface

The annotation looks something like this:

```yaml
cdi.k8s.io/gpu: nvidia.com/pgpu=0
```

The key consists of the mandatory CDI prefix `cdi.k8s.io`, followed by an arbitrary path component, in this case `gpu`.
The value specifies a device kind, `nvidia.com/pgpu`, and the name of the desired device in the CDI spec, `0`.

The CDI spec is assembled from files in well-known locations (`/etc/cdi`, `/run/cdi`) that look like this:

```yaml
kind: nvidia.com/pgpu
cdiVersion: 0.5.0
devices:
- name: "0"
  containerEdits:
    deviceNodes:
    - path: /dev/vfio/54
    # other possible keys: env, mounts, hooks, etc.
```

Each file declares a device kind and a list of named devices of this kind.
The individual devices come with a set of modifications that need to be applied to the container in order to make the device available.
That includes adding the device itself, but also shared libraries and configuration that are required to work with this device.

## Kubernetes scheduler

The scheduler sums up the extended resources on a pod and looks for a node that has sufficient capacity.
If successful, it assigns the pod to that node.
Besides that, the scheduler neither deals with device allocation, nor with CDI annotations.

## Kubelet

The Kubelet sees the extended resource on the pod and tries to find a device plugin for this device class.
The device plugin receives an `Allocate` request and returns a list of container edits, similar to those in CDI.
For now, resources are per-container, which means that the Kubelet will only allocate a device for the container with the extended resource request.

In our example above, this means that the container `main` will receive two additional devices: `/dev/vfio/vfio` and `/dev/vfio/54`.
This is problematic: the VM will be created before the first container (pause), but GPUs can only be cold-plugged into a CVM and thus need to be known at VM creation time.
This is where the CDI annotation comes in, but only after passing through containerd.

## containerd

containerd supports CDI natively, as outlined in the CDI README.
However, we don't enable it because it would interfere with the CDI handler in the Kata runtime.
<!-- TODO(burgerdev): this is not documented anywhere. -->
Suffice to say, containerd passes the CDI annotation and, in the case of the main container, the device to the Kata shim.

## Kata runtime

When the Kata runtime receives a `CreateSandbox` request, it [looks specifically for CDI annotations](https://github.com/kata-containers/kata-containers/blob/c75a46d17f800e1d825aca31c62c7bf3f44ca8b1/src/runtime/pkg/containerd-shim-v2/create.go#L111-L125).
This results in the pause container receiving the same device edits as the main container did, but from the CDI spec instead of the device plugin.
With this information, Kata can set up the VM with the GPU plugged in.
It also [stores a list of devices attached to the VM](https://github.com/kata-containers/kata-containers/blob/c75a46d17f800e1d825aca31c62c7bf3f44ca8b1/src/runtime/virtcontainers/container.go#L1059), linking the host device path with its corresponding guest PCI port.

For every regular container, the Kata agent checks the devices in the OCI spec against the list of attached PCIe devices.
If there's a match, it [adds a _sibling annotation_](https://github.com/kata-containers/kata-containers/blob/c75a46d17f800e1d825aca31c62c7bf3f44ca8b1/src/runtime/virtcontainers/container.go#L1098-L1111).
This annotation anticipates the CDI spec generated within the guest, based on the assigned port.
The _outer annotations_, which were consumed above, are removed in order to not confuse the runtime.

If you payed close attention, you might have noticed the following:

* The CDI annotation requests a _specific_ resource, identified by vendor, class and name.
* The extended resource requests an _anonymous_ resource by vendor and class.

This soon leads to problems, for example when there are multiple GPUs on a single node.
If the device plugin allocates a GPU that's not the one requested with the CDI annotation, the sibling matching in its current form won't work.
<!-- TODO(burgerdev): maybe it could work ... -->

## Guest VM

After boot, the NVIDIA Container Toolkit (CTK) scans the available PCI devices for NVIDIA cards.
For each card found, it adds an appropriate entry to the guest's CDI spec.
This all happens before the Kata agent starts, so that the CDI spec is ready when the first container is started.

## Kata agent

The Kata agent receives `CreateContainer` requests and inspects their annotations.
Each CDI annotation is passed to the CDI library, which extends the OCI runtime config according to the CDI spec generated by the NVIDIA CTK.
Here, the journey ends with containers equipped with their expected devices.

## Outlook

The system we're looking at is brittle and a bit counterintuitive: where do we go from here?
There are a few proposals that might drive this integration forward:

1. A [KEP](https://github.com/kubernetes/enhancements/pull/4113) that would allow the CRI (and, thus, hopefully the Kata runtime) to know about requested resources at sandbox creation time.
2. A [proposal](https://github.com/kata-containers/kata-containers/issues/12009) to let the Kata runtime collaborate with the device plugin.
