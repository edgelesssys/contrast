![Contrast](docs/static/img/banner.svg)

# Contrast

Contrast runs confidential container deployments on Kubernetes at scale.

Contrast is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects.
Confidential Containers are Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation from the surrounding environment.
This works with unmodified containers in a lift-and-shift approach.
Contrast currently targets the [CoCo preview on AKS](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview).

## Goal

Contrast is designed to keep all data always encrypted and to prevent access from the infrastructure layer. It removes the infrastructure provider from the trusted computing base (TCB). This includes access from datacenter employees, privileged cloud admins, own cluster administrators, and attackers coming through the infrastructure, for example, malicious co-tenants escalating their privileges.

Contrast integrates fluently with the existing Kubernetes workflows. It's compatible with managed Kubernetes, can be installed as a day-2 operation and imposes only minimal changes to your deployment flow.

## Use Cases:

* Increasing the security of your containers
* Moving sensitive workloads from on-prem to the cloud with Confidential Computing
* Shielding the code and data even from the own cluster administrators
* Increasing the trustworthiness of your SaaS offerings
* Simplifying regulatory compliance
* Multi-party computation for data collaboration

## Features

### 🔒 Everything always encrypted

* Runtime encryption: All Pods run inside AMD SEV-based Confidential VMs (CVMs). Support for Intel TDX will be added in the future.
* PKI and mTLS: All pod-to-pod traffic can be encrypted and authenticated with Contrast's workload certificates.

### 🔍 Everything verifiable

* Workload attestation based on the identity of your container and the remote-attestation feature of [Confidential Containers](https://github.com/confidential-containers)
* "Whole deployment" attestation based on Contrast's [Coordinator attestation service](#the-contrast-coordinator)
* Runtime environment integrity verification based runtime policies
* Kata micro-VMs and single workload isolation provide a minimal Trusted Computing Base (TCB)

### 🏝️ Everything isolated

* Runtime policies enforce strict isolation of your containers from the Kubernetes layer and the infrastructure.
* Pod isolation: Pods are isolated from each other.
* Namespace isolation: Contrast can be deployed independently in multiple namespaces.

### 🧩 Lightweight and easy to use

* Install in Kubernetes cluster as a day-2 operation.
* Compatible with managed Kubernetes.
* Minimal DevOps involvement.
* Simple CLI tool to get started.

## Components

### The Contrast Coordinator

The Contrast Coordinator is the central remote attestation service of a Contrast deployment.
It runs inside a confidential container inside your cluster.
The Coordinator can be verified via remote attestation, and a Contrast deployment is self-contained.
The Coordinator is configured with a *manifest*, a configuration file containing the reference attestation values of your deployment.
It ensures that your deployment's topology adheres to your specified manifest by verifying the identity and integrity of all confidential pods inside the deployment.
The Coordinator is also a certificate authority and issues certificates for your workload pods during the attestation procedure.
Your workload pods can establish secure, encrypted communication channels between themselves based on these certificates and the Coordinator as the root CA.
As your app needs to scale, the Coordinator transparently verifies new instances and then provides them with their certificates to join the deployment.

To verify your deployment, the Coordinator's remote attestation statement combined with the manifest offers a concise single remote attestation statement for your entire deployment.
A third party can use this to verify the integrity of your distributed app, making it easy to assure stakeholders of your app's identity and integrity.

### The Manifest

The manifest is the configuration file for the Coordinator, defining your confidential deployment.
It's automatically generated from your deployment by the Contrast CLI.
It currently consists of the following parts:

* *Policies*: The identities of your Pods, represented by the hashes of their respective runtime policies.
* *Reference Values*: The remote attestation reference values for the Kata confidential micro-VM that's the runtime environment of your Pods.
* *WorkloadOwnerKeyDigest*: The workload owner's public key digest. Used for authenticating subsequent manifest updates.

### Runtime Policies

Runtime Policies are a mechanism to enable the use of the untrusted Kubernetes API for orchestration while ensuring the confidentiality and integrity of your confidential containers.
They allow us to enforce the integrity of your containers' runtime environment as defined in your deployment files.
The runtime policy mechanism is based on the Open Policy Agent (OPA) and translates the Kubernetes deployment YAML into the Rego policy language of OPA.
The Kata Agent inside the confidential micro-VM then enforces the policy by only acting on permitted requests.
The Contrast CLI provides the tooling for automatically translating Kubernetes deployment YAML into the Rego policy language of OPA.

The trust chain goes as follows:

1. The Contrast CLI generates a policy and attaches it to the pod definition.
2. Kubernetes schedules the pod on a node with the confidential computing runtime.
3. Containerd takes the node, starts the Kata Shim and creates the pod sandbox.
4. The Kata runtime starts a CVM with the policy's digest as `HOSTDATA`.
5. The Kata runtime sets the policy using the `SetPolicy` method.
6. The Kata agent verifies that the incoming policy's digest matches `HOSTDATA`.
7. The CLI sets a manifest in the Contrast Coordinator, including a list of permitted policies.
8. The Contrast Coordinator verifies that the started pod has a permitted policy hash in its `HOSTDATA` field.

After the last step, we know that the policy hasn't been tampered with and, thus, that the workload is as intended.

### The Contrast Initializer

Contrast provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate. The Initializer runs as an init container before your workload is started.

### The Contrast runtime

Contrast depends on a Kubernetes [runtime class](https://kubernetes.io/docs/concepts/containers/runtime-class/), which is installed
by the `node-installer` DaemonSet.
This runtime consists of a containerd runtime plugin, a virtual machine manager (cloud-hypervisor), and a podvm image (IGVM and rootFS).
The installer takes care of provisioning every node in the cluster so it provides this runtime class.

## Current limitations

Contrast is in an early preview stage, and most underlying projects are still under development as well.
As a result, there are currently certain limitations from which we try to document the most significant ones here:

- Only available on AKS with CoCo preview (AMD SEV-SNP)
- Persistent volumes currently not supported in CoCo
- While workload policies are functional in general, but [not covering all edge cases](https://github.com/microsoft/kata-containers/releases/tag/genpolicy-0.6.2-5)
- Port-forwarding isn't supported by Kata Containers yet
- CLI is only available for Linux (mostly because upstream dependencies aren't available for other platforms)
- Known bugs and limitations on AKS CoCo
  * The total amount of container image layers per pod is restricted to 32.
  * Container memory limits are wrongly applied. Workaround: only use memory requests.
  * Directories with a large number of files may cause applications to hang. Workarounds:
    - During image build, try to keep directories under 4096 bytes (~200 files).
    - At runtime, `touch` a file in the affected directory to force it into the `overlayfs` working directory.

## Upcoming Contrast features

- Transparent service mesh (apps can currently use mTLS with Coordinator certs for secure communication)
- Plugin for a key management service (KMS) for attestation/coordinator certificate-based key release
- High availability (distributed Contrast Coordinator)

## Contributing

See the [contributing guide](CONTRIBUTING.md).
Please follow the [Code of Conduct](/CODE_OF_CONDUCT.md).

## Support

* If something doesn't work, make sure to use the [latest release](https://github.com/edgelesssys/contrast/releases/latest) and check out the [known issues](https://github.com/edgelesssys/contrast/issues?q=is%3Aopen+is%3Aissue+label%3A%22known+issue%22).
* Please file an [issue](https://github.com/edgelesssys/contrast/issues) to get help or report a bug.
* Visit our [blog](https://www.edgeless.systems/blog/) for technical deep-dives and tutorials and follow us on [LinkedIn](https://www.linkedin.com/company/edgeless-systems) for news.
* Edgeless Systems also offers [Enterprise Support](https://www.edgeless.systems/products/contrast/).
