# Nunki

Nunki ([/ˈnʌŋki/](https://en.wikipedia.org/wiki/Sigma_Sagittarii)) runs confidential container deployments
on untrusted Kubernetes at scale.

Nunki is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects. Confidential Containers are
Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation
from the surrounding environment. This works with unmodified containers in a lift-and-shift approach.

## The Nunki Coordinator

The Nunki Coordinator is the central remote attestation component of a Nunki deployment. It's a certificate
authority and issues certificates for workload pods running inside confidential containers. The Coordinator
is configured with a *manifest*, a configuration file that holds the reference values of all other parts of
a deployment. The Coordinator ensures that your app's topology adheres to your specified manifest. It verifies
the identity and integrity of all your services and establishes secure, encrypted communication channels between
the different parts of your deployment. As your app needs to scale, the Coordinator transparently verifies new
instances and then provides them with mesh credentials.

To verify your deployment, the remote attestation of the Coordinator and its manifest offers a single remote
attestation statement for your entire deployment. Anyone can use this to verify the integrity of your distributed
app, making it easier to assure stakeholders of your app's security.

## The Nunki Initializer

Nunki provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate. The Initializer runs as init container before your workload is started.

## Generic Workflow

### Setup

First of all, you will need a Nunki binary and a workspace directory. At this stage of the project,
you will need to follow the instructions in [CONTRIBUTING.md](CONTRIBUTING.md) to set up a
development environment and then run `just demodir` to initialize a workspace. In the future, this
step may become part of the released Nunki binary.

### Kubernetes Resources

In this step, you will define Kubernetes resources for the Nunki coordinator and the workloads you
want to run. Nunki works with vanilla Kubernetes yaml, so all you have to do is copy the resource
files into a `resources` sub-directory of your workspace. If you use Helm, you can do that with
`helm template`; if you use Kustomize, you can do that with `kustomize build`.

All pod definitions and templates in the resources need an additional init container that talks to
the coordinator and eventually obtains a certificate to use for mesh communication between
confidential workloads. The definition can be taken directly from
[deployments/simple/initializer.yml](deployments/simple/initializer.yml). Furthermore, the runtime
class needs to be set to `kata-cc-isolation` so that the workloads are started as confidential
containers.

Finally, you will need to deploy the Nunki coordinator, too. Copy the deployment from
[deployments/simple/coordinator.yml](deployments/simple/coordinator.yml) and adjust it as you see
fit (e.g. labels, namespace, service attributes).

### Generate Policies and Manifest

With all your Kubernetes resource definitions in a directory, the next step is to define
*Policies* for them.
<!-- TODO: cross-reference to definition -->
The policy for a pod specifies what image the container runtime is allowed start, and what the
configuration for the container may or may not look like. Fortunately, this process is fully
automated given the resource definitiions.

In your workspace directory, run `./nunki generate resources/*` to annotate all resources with a
policy. It takes some time to execute because it needs to download the desired container images for
inspection. Finally, it writes a *Manifest* file to the current directory.
<!-- TODO: cross-reference to definition -->

The manifest maps each workload to the corresponding execution policy (identified by its digest).
During attestation, the coordinator ensures that the workload is governed by this policy.
Furthermore, the manifest contains the reference values for various confidential computing
attributes, such as platform versions and the Azure MAA keys.

### Apply Resources

With the policy annotations in place, we are now ready to deploy. In your workspace directory, run
`kubectl apply -f resources/`. This will launch the coordinator and your workloads in the AKS
cluster.

During startup, the coordinator creates TLS certificates - a root certificate and an intermediate
certificate for signing workload certificates. The other workloads will stay in the initialization
phase until the manifest is published to the coordinator.

For the next steps, we will need to connect to the coordinator. If you did not expose the
coordinator with an external Kubernetes LoadBalancer, you can expose it locally with
`kubectl port-forward`. We're going to assume that the coordinator is exposed on `localhost:1313`
in the following sections.

### Set Manifest

The coordinator needs to know what workloads it is supposed to trust. This information is part of
the manifest we generated earlier. To publish the manifest, you run
`./nunki set -c localhost:1313` in your workspace directory. This command establishes an attested
TLS connection (aTLS) to the coordinator and sets the manifest. The coordinator proceeds to issue
certificates to those workloads that match a policy, and the regular containers start running.

### Verify the Coordinator

Before we can share sensitive information with the workloads running in AKS, we need to verify
them. The chain of trust is:

1. The coordinator verifies that the workload is running according to policy, and only issues a
   certificate if that's the case.
2. We verify that the coordinator is running according to the coordinator policy, that the expected
   manifest is set and that it's in possession of the intermediate certificate's private key.

Thus, we know that every workload that presents a Nunki mesh certificate is trustworthy. To proceed
with verification, run `./nunki verify -c localhost 1313` in your workspace directory. This will
create a `verify/` sub-directory containing the manifest history and the TLS certificates.

### Communicate with Workloads

Finally, we're all set. The workloads should have obtained certificates from the coordinator, and
its time to put them to use. Assuming your workload is exposed at `localhost:8443` (e.g. with
`kubectl port-forward`), you can establish a TLS connection by configuring the coordinator's mesh
root as a trusted CA certificate. For example, with `curl` you would run:

```sh
curl --cacert ./verify/mesh-root.pem https://localhost:8443
```

## Contributing

See the [contributing guide](CONTRIBUTING.md).
