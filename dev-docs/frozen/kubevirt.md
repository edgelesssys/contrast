# KubeVirt

Contrast, as of today, is limited to confidential containers.
The only reason for this limitation is the focus on the Kata runtime: Coordinator and manifest would theoretically support other CVMs.
If there was a runtime for CVMs, we could adopt it and build out support for its CVMs in our attestation packages.

KubeVirt is a widely used project that manages bare metal CVMs based on Kubernetes CRDs.
It recently released support for confidential VMs, which made it a candidate for a Contrast CVM runtime.
Thus, we evaluated the current state of KubeVirt and how we could schedule CVMs with it.

## Setup

We installed KubeVirt [`v1.8.4`](https://github.com/kubevirt/kubevirt/releases/tag/v1.8.4) with a kustomization like this:

<details>
<summary>Kustomization to enable SEV feature gate</summary>

```yaml
resources:
- https://github.com/kubevirt/kubevirt/releases/download/v1.8.4/kubevirt-operator.yaml
- https://github.com/kubevirt/kubevirt/releases/download/v1.8.4/kubevirt-cr.yaml

apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

patches:
- patch: |-
    - op: replace
      path: /spec/configuration/developerConfiguration/featureGates
      value: ["WorkloadEncryptionSEV"]
  target:
    group: kubevirt.io
    kind: KubeVirt
    name: kubevirt
    version: v1
- patch: |-
    - op: add
      path: /metadata/labels/ci.contrast.edgeless.systems~1keep
      value: "true"
  target:
    kind: Namespace
    name: kubevirt
```

</details>

We also installed the `virtctl` binary to access the serial console of VMs.

## Notable design choices

- KubeVirt uses CRDs to manage the lifecycle of VMs.
- Required data (VM images, kernels, ...) is usually provided as OCI artifacts.
- Another notable design is the deep integration of [`cloud-init`](https://docs.cloud-init.io/en/26.1/).

## Running CVMs

We used the example from the [official docs](https://kubevirt.io/user-guide/cluster_admin/confidential_computing/#deploying-amd-sev-snp-enabled-vms).

<details>
<summary>`VirtualMachineInstance` CR for SNP</summary>

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  labels:
    special: vmi-fedora
  name: vmi-fedora
spec:
  domain:
    launchSecurity:
      snp: {}
    firmware:
      bootloader:
        efi:
          secureBoot: false
    devices:
      disks:
      - disk:
          bus: virtio
        name: containerdisk
      - disk:
          bus: virtio
        name: cloudinitdisk
      rng: {}
    resources:
      requests:
        memory: 1024M
  terminationGracePeriodSeconds: 0
  volumes:
  - containerDisk:
      image: registry:5000/kubevirt/fedora-with-test-tooling-container-disk:devel
    name: containerdisk
  - cloudInitNoCloud:
      userData: |-
        #cloud-config
        password: fedora
        chpasswd: { expire: False }
    name: cloudinitdisk
```

</details>

First notable finding is that the example uses an image that's not public.
We tried replacing the registry with `quay.io`, assuming that the image published there is the same one.

The VMI reported as running, but we couldn't access the serial console.
The virt-launcher pod shows an error in the guest console log, pointing to early boot issues:

```txt
BdsDxe: failed to load Boot0002 "Grub Bootloader" from Fv(7CB8BDC9-F8EB-4F34-AAEA-3EE4AF6516A1)/FvFile(B5AE312C-BC8A-43B1-9C62-EBB826DD5D07): Not Found
BdsDxe: No bootable option was found.
```

It looks like the VM image expects GRUB, but doesn't find it.
Most likely, the example VM image from the OCI artifact isn't compatible with the SNP boot chain.
We [asked for clarification](https://kubernetes.slack.com/archives/C0163DT0R8X/p1782318623026589), but didn't get a response.

## Other observations

The KubeVirt SNP implementation is rather simplistic.
It's not configurable at all, the [API object](https://kubevirt.io/api-reference/v1.8.4/definitions.html#_v1_sevsnp) is just empty.
We also noted that it's not possible to bring a custom firmware, per the [`Firmware`](https://kubevirt.io/api-reference/v1.8.4/definitions.html#_v1_firmware) and [`EFI`](https://kubevirt.io/api-reference/v1.8.4/definitions.html#_v1_efi) objects.

## Conclusion

While promising at first glance, KubeVirt doesn't seem ready for bringing CVMs to Contrast.
Even if there was a working example, we can't configure the CVM with launch policy, IDBlock, etc.
Using KubeVirt at this point would require a non-neglible amount of work in a fork.
