# Policies

Kata runtime policies are an integral part of the Confidential Containers preview on AKS.
They prescribe how a Kubernetes pod must be configured to launch successfully in a confidential VM.
In Contrast, policies act as a workload identifier: only pods with a policy registered in the manifest receive workload certificates and may participate in the confidential deployment.
Verification of the Contrast Coordinator and its manifest transitively guarantees that all workloads meet the owner's expectations.

## Structure

The Kata agent running in the confidential micro-VM exposes an RPC service [`AgentService`] to the Kata runtime.
This service handles potentially untrustworthy requests from outside the TCB, which need to be checked against a policy.

Kata runtime policies are written in the policy language [Rego].
They specify what `AgentService` methods can be called, and the permissible parameters for each call.

Policies consist of two parts: a list of rules and a data section.
While the list of rules is static, the data section is populated with information from the `PodSpec` and other sources.

[`AgentService`]: https://github.com/kata-containers/kata-containers/blob/e5e0983/src/libs/protocols/protos/agent.proto#L21-L76
[Rego]: https://www.openpolicyagent.org/docs/latest/policy-language/

## Generation

Runtime policies are programmatically generated from Kubernetes manifests by the Contrast CLI.
The `generate` subcommand inspects pod definitions and derives rules for validating the pod at the Kata agent.
There are two important integrity checks: container image checksums and OCI runtime parameters.

For each of the container images used in a pod, the CLI downloads all image layers and produces a cryptographic [dm-verity] checksum.
These checksums are the basis for the policy's _storage data_.

The CLI combines information from the `PodSpec`, `ConfigMaps`, and `Secrets` in the provided Kubernetes manifests to derive a permissible set of command-line arguments and environment variables.
These constitute the policy's _OCI data_.

[dm-verity]: https://www.kernel.org/doc/html/latest/admin-guide/device-mapper/verity.html

## Evaluation

The generated policy document is annotated to the pod definitions in Base64 encoding.
This annotation is propagated to the Kata runtime, which calculates the SHA256 checksum for the policy and uses that as SNP `HOSTDATA` or TDX `MRCONFIGID` for the confidential micro-VM.

After the VM launched, the runtime calls the agent's `SetPolicy` method with the full policy document.
If the policy doesn't match the checksum in `HOSTDATA` or `MRCONFIGID`, the agent rejects the policy.
Otherwise, it applies the policy to all future `AgentService` requests.

## Guarantees

The policy evaluation provides the following guarantees for pods launched with the correct generated policy:

- Command and its arguments are set as specified in the resources.
- There are no unexpected additional environment variables.
- The container image layers correspond to the layers observed at policy generation time.
  Thus, only the expected workload image can be instantiated.
- Executing additional processes in a container is prohibited.
- Sending data to a container's standard input is prohibited.

The current implementation of policy checking has some blind spots:

- Containers can be started in any order, or be omitted entirely.
- Environment variables may be missing.
- Volumes other than the container root volume don't have integrity checks (particularly relevant for mounted `ConfigMaps` and `Secrets`).

## Trust

Contrast verifies its confidential containers following these steps:

1. The Contrast CLI generates a policy and attaches it to the pod definition.
2. Kubernetes schedules the pod on a node with the confidential computing runtime.
3. Containerd invokes the Kata runtime to create the pod sandbox.
4. The Kata runtime starts a CVM with the policy's digest as `HOSTDATA`/`MRCONFIGID`.
5. The Kata runtime sets the policy using the `SetPolicy` method.
6. The Kata agent verifies that the incoming policy's digest matches `HOSTDATA`/`MRCONFIGID`.
7. The CLI sets a manifest in the Contrast Coordinator, including a list of permitted policies.
8. The Contrast Initializer sends an attestation report to the Contrast Coordinator, asking for a mesh certificate.
9. The Contrast Coordinator verifies that the started pod has a permitted policy hash in its `HOSTDATA`/`MRCONFIGID` field.

After the last step, we know that the policy hasn't been tampered with and, thus, that the workload matches expectations and may receive mesh certificates.

## Platform Differences

Contrast uses different rules and data sections for different platforms.
This results in different policy hashes for different platforms.
The `generate` command automatically derives the correct set of rules and data sections from the `reference-values` flag.

## Supported resource kinds

Contrast policies can be generated for all Kubernetes resource kinds that spawn pods, including:

<!-- keep-sorted start by_regex=`(\w+)` -->
- `CronJob`
- `DaemonSet`
- `Deployment`
- `Job`
- `Pod`
- `ReplicaSet`
- `ReplicationController`
<!-- keep-sorted end -->

The following resource kinds influence pod configuration and are also taken into account for policy generation:

<!-- keep-sorted start by_regex=`(\w+)` -->
- `ClusterRole`
- `ClusterRoleBinding`
- `ConfigMap`
- `LimitRange`
- `PodDisruptionBudget`
- `Role`
- `RoleBinding`
- `Secret`
- `Service`
- `ServiceAccount`
<!-- keep-sorted end -->

All other resource kinds are unsupported and can't be passed to `contrast generate`.
