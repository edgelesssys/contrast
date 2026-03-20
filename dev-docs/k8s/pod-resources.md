# Pod resources

## Background

There has been some confusion around the use of memory limits in Kata.
In this doc, you'll find some pointers to how pod resource limits are determined by Kubernetes and how they're implemented in Kata.

## How memory limits are calculated

* Entrypoint: `ResourceConfigForPod` in <https://github.com/kubernetes/kubernetes/blob/v1.31.5/pkg/kubelet/cm/helpers_linux.go#L121>.
* Formula: `PodLimits` in <https://github.com/kubernetes/kubernetes/blob/v1.31.5/pkg/api/v1/resource/helpers.go#L152>.

## How memory limits are propagated

From Kubelet to containerd via CRI method `RunPodSandbox`:

```go
cri.RunPodSandboxRequest{
    Config: &cri.PodSandboxConfig{
        Linux: &cri.LinuxPodSandboxConfig{
            Resources: &cri.LinuxContainerResources{
                MemoryLimitInBytes: 1234,
            },
        },
    },
}
```

From containerd to Kata via sandbox container (that is, pause container) annotation: <https://github.com/containerd/containerd/blob/v2.0.0/internal/cri/server/podsandbox/sandbox_run_linux.go#L186>.

Kata converts the annotation to MiB and uses it to calculate the VM size: <https://github.com/kata-containers/kata-containers/blob/3.18.0/src/runtime/pkg/oci/utils.go#L1391>.

## Examples

`default_memory` in our config is 512, the experiments were conducted from <https://github.com/edgelesssys/contrast/commits/a0021ad6b32afaa45b031ba015087de8d7502c8e>.

| Main Container Limit | Init Container Limit | Sidecar Container Limit | Annotation Value | QEMU Parameter |
| -------------------- | -------------------- | ----------------------- | ---------------- | -------------- |
| 100 MB               | 200 MB               | —                       | 209715200        | `-m 712M`      |
| 100 MB               | —                    | 200 MB                  | 314572800        | `-m 812M`      |
| 100 MB               | —                    | No limit                | 104857600        | `-m 612M`      |
