# Security Overview

## Architectual goals

- Isolation: Your cloud workloads and processed data is inaccessible to the infrastructure provider. This is achieved by running pods within CVMs with runtime encryption and encrypting data whenever it leaves the CVM. Workload and data access can also be forbidden for the Kubernetes admins through a fine-grained access policy.
- Attestation: The integrity of your workloads is ensured through hardware-enforced and provider-independent attestation.
- Integration: Contrast offers a custom Kubernetes runtime CVMs and specialized workloads that can be integrated into your existing cluster with only minimal adjustments of the development YAMLs.

## Key components

- Contrast Kubernetes Runtime: A custom Kuberenetes RuntimeClass specifices the woker node's runtime which creates a specialized host enviroment and ensures that pods run inside CVMs on worker nodes.
- Contrast Coordinator: A workload added to you cluster. It runs within a CVM and serves as the central attestation service only including attested workloads within a trusted PKI.
- Manifest: Configured within the Coordinator, the manifest serves as the reference for a trusted state of your cluster, specifying cryptographic hashes for all your application workloads. Attestation of CVMs are always checked against this manifest.
- Contrast CLI: Runs on the workload operator's machine and serves as the primary management tool for Contrast deployments. It provides tools to verify the integrity and authenticity of the Coordinator and the entire deployment via remote attestation.

## Attestation

## Service mesh

### Recovery

### Configuration

## Integration Flow

### Setting up your cluster

You can set up your cluster using k3s or apply contrast to your already existing cluster. It just has to be confidential-computing-ready and [some adjustments](https://docs.edgeless.systems/contrast/getting-started/bare-metal?vendor=amd) to the nodes have to be made.

### Setting up worker node's hosts

Each worker node sets up a host environment to run pods in CVMs.

1. Uninitialized nodes joining the cluster have the following installed (through k3s):
   1. Operating System
   2. Kubelet
   3. Basic Container runtime `containerd` (to run `DaemonSet`)
2. A custom RunTime handler (defined as `RunTimeClass`) and `DaemonSet` (node installer) deployment is applied to Kubernetes.
   - defined in deployment file: https://github.com/edgelesssys/contrast/releases/download/v1.5.1/runtime-k3s-qemu-snp.yml
   - The custom `DaemonSet` is used to set up the host in the node
   - The custom runtime handler is used for creating confidential VMs for running workload pods.
3. Kubelet running on uninitialized nodes continously listens to updates for pods to run
4. Kubelet fetches new pod configuration from the Kubernetes API and uses `containerd` to fetch the `DaemonSet` pod.
5. `DaemonSet`sets up the host environment of the node:
   1. Install the Contrast containerd shim (`containerd-shim-contrast-cc-v2`)
   2. Install `cloud-hypervisor` or `QEMU` as the virtual machine manager (VMM)
   3. Install an IGVM file or separate firmware and kernel files for pod-VMs of this class
   4. Install a read-only root filesystem disk image for the pod-VMs of this class
   5. Reconfigure `containerd` by adding a runtime plugin that corresponds to the `handler` field of the Kubernetes `RuntimeClass`
   6. Restart `containerd` to make it aware of the new plugin

After these steps the host system is ready to set up the CVM where your actual pod is running.

### Setting up CVM

1. The installed runtime plugin `containerd.kata-cc.v2` takes over to set up the CVM using the hypervisor `QEMU`.

- Here, the `QEMU`command line is generated including options for `HOSTDATA` injection
- `HOSTDATA`contains the SHA256 hash of the runtime policy
- while not part of the initial memory, `HOSTDATA` is and hardware-enforced field and is always included and signed in the attestation report.

2. The plugin instructs `QEMU` to boot a confidential VM (CVM) based on the IGVM file
3. `QEMU` starts the CVM, loads the kernel and mounts `initrd` by considering parameters from the kernel command line
   - kernel command line includes explicit instructions for boot, init process, and filesystem mounts. It also includes the dm-verity hash for the integrity enforcement of the root filesystem.
   - the kernel includes the `dm-verity` drivers.
   - `QEMU`injects HOSTDATA into the CVM (via `SEV-SNP LAUNCH_SECRET`insruction, storing it inside AMD's hardware protected area).
4. `initrd` as a minimal filesystem runs scripts that actually mount the rootfs via `dm-verity`
5. `initrd` hands off to `systemd` in the rootfs
6. `kata-agent` is started.
7. `SetPolicy`of the `kata-agent`is called which reads the full policy document and compares the freshly generated hash with the one in `HOSTDATA`. If no match, the CVM aborts execution.

### Setting up workload

The `kata-agent` within the CVM is responsible for launching workloads inside the guest (CVM).

1. The `nydus-snapshotter` plugin of containerd pulls metadata for containers to launch within the CVM and forwards this to th `kata-agent`
2. The `kata-agent` pulls the images and compares the hashes with the ones configured in the policy.
3. If equal, the `kata-agent` runs the container.

### Contrast-specific containers on worker nodes

Within every CVM Pod two additional containers are started before the applocation container:

1. **Init Container**: Runs to completion and is responsible for issuing a service mesh-certificate to enter the cluster internal PKI managed by the Contrast Coordinator.
2. **Sidecar Container**: A secondary container running along the application container. In case of Contrast, it serves a proxy that wraps network traffic inside mutual TLS (mTLS) based on the service-mesh certificates. It is enabled via annotations in the deployment YAML.
   - iptable rules ensure that all traffic is routed through the proxy
   - Envoy handles mTLS for both incoming (ingress) and outgoing (egress) traffic.
   - A Transparent Proxy (TP) means the application don't need to be modified.

### Workload enters Contrast service mesh

1. The Init container running inside the CVM pod generates a private, public key pair. The private key never leaves the CVM.
2. The Init container initializes an issuing of a service-mesh certificate from the Coordinator.
3. The Coordinator sets up an aTLS connection to the Init Container
   - Only if attestation of the worker is successful a TLS connection will be established and a valid service-mesh certificate is issued.
4. The Coordinator sends the service-mesh certificate to the Init container of the workload along with a secret seed for recovery
5. The Init container uses this seed to derive encryption keys and LUKS encryption to encrypt the secret service-mesh key to untrusted persitent volumes.

## Verifying cluster integrity

The User uses the CLI to verify the integrity and confidentiality of the Kubernetes cluster.

1. The CLI is released with reference hashes (launch digest, policy hash) included. These reference values can be generated and checked by a client trough reproducible builds.
2. The CLI establishes an aTLS connection to the Coordinator
   - The TLS protocol is extended to also check the attestation report of the Coordinator against its embedded reference values.
   - If attestation is successfull, CLI establishes an aTLS connection.
3. The CLI annotates your YAML deployments and generates a manifest via `generate` command
4. The CLI uploades the manifest to the coordinator via `set` command.
   - This serves as reference for the trusted state of your deployment
5. The CLI can verify the currently configured manifest by using the `verify` command and checking against a given manifest.
   - This checks if the enforced state is the one expected by a user.

## Deployment updates
