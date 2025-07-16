# Kata Agent Policy

## Background

Kata Containers is an [OCI Runtime] and implements the [Containerd ShimV2 API].
Both APIs are fundamentally container-centric and not concerned with the concept of pods or container image layers.
A CRI implementation is necessary to translate Kubernetes artefacts into container runtime API calls.
In the case of CoCo, this is done by containerd.

The Kata Runtime actually consists of two parts:

1. The runtime implementation runs on the Kubernetes node.
2. The Kata Agent runs in the confidential guest.

The runtime and the agent communicate over [VSOCK], exchanging [AgentService] messages.

[OCI Runtime]: https://github.com/opencontainers/runtime-spec
[Containerd ShimV2 API]: https://pkg.go.dev/github.com/containerd/containerd@v1.7.13/api/runtime/task/v2#TaskService
[VSOCK]: https://www.man7.org/linux/man-pages/man7/vsock.7.html
[AgentService]: https://github.com/kata-containers/kata-containers/blob/89c76d7/src/libs/protocols/protos/agent.proto#L21-L76

## Trust

In CoCo, the agent is part of the TEE but the runtime isn't.
To trust the agent, we need to ensure that the agent only serves permitted requests.
For Contrast, the chain of trust looks like this:

1. The CLI generates a policy and attaches it to the pod definition.
2. Kubernetes schedules the pod on a node with a CoCo runtime.
3. Containerd takes the node, starts the Kata Shim and creates the pod sandbox.
4. The Kata runtime starts a CVM with the policy's digest as `HOSTDATA`.
5. The Kata runtime sets the policy using the `SetPolicy` method.
6. The Kata agent verifies that the incoming policy's digest matches `HOSTDATA`.
7. The CLI sets a manifest at the Contrast Coordinator, including a list of permitted policies.
8. The Contrast Coordinator verifies that the started pod has a permitted policy hash in its `HOSTDATA` field.

After the last step, we know that the policy hasn't been tampered with and thus that the workload is as intended.

## Policy Structure

The policy is written in [Rego] and consists of *rules* and *data*.

The rules are somewhat static - in case of Contrast, they're baked into the CLI.
The upstream tool `genpolicy` supports an additional settings file to augment the rules with site-specific information.

The data section is specific to the pod at hand and is generated from the deployment YAML.

Next to this document, you can find a [pod definition](example-policy.yml) and the corresponding [generated policy](example-policy.rego).
The policy was created with `nix run .#cli-release` at commit `6d25a1b4c82adeb4fff2771453bc38ca44cde466`.

[Rego]: https://www.openpolicyagent.org/docs/latest/policy-language/

## Policy Evaluation

There is a matching rule for each `AgentService` method, although some of them are just blanket allow or deny.
Most interesting for us is the rule for `CreateContainer`.
It does some general sanity checks, and then compares the data in the `CreateContainerRequest` with the data in the policy.

## Policy Rules

The rules can be divided into two major checks: *OCI spec* and *storage*.

### OCI Rules

The OCI spec check is concerned with the content of the [OCI config] requested by the Kubelet.
This includes command line arguments, environment variables and security configuration.

[OCI config]: https://github.com/opencontainers/runtime-spec/blob/cb7ae92/specs-go/config.go#L6-L34

### Storage Rules

The storage check is concerned with the integrity of the various mount points for the container.
Of particular interest is the container's root filesystem.
The host's containerd snapshot plugin pulls the image layer tarballs.
These are published to the guest as block devices, which the guest then maps with dm-verity, mounts as tarFS and combines into an overlayFS.
The expected verity hashes are part of the policy data, the actual hashes are injected into the request.

TODO(burgerdev): discuss ConfigMaps, ephemeral mounts, etc.

## Policy Generation

Policies are generated with the [`genpolicy` tool] from local Kubernetes resources.
The tool analyzes the `PodSpec` of pods, deployments, etc., anticipates the corresponding Kata Runtime requests and creates request template data accordingly.
In addition to the Kubernetes resources, the tool expects two input files: rules and settings.
The settings customize some aspects of policy generation (mostly CRI defaults) which are added to the request template.
The request template data is appended to the rules file, and together they form an executable policy.

[`genpolicy` tool]: https://github.com/kata-containers/kata-containers/tree/main/src/tools/genpolicy

## Policy Evaluation and Debugging

The only practical way to debug policy decisions right now is to look at OPA logs.
These logs are included in the containerd error message and need to be extracted first:

```sh
kubectl events --for pod/${failing_pod} -o json |
  jq -r '.items[-1].message' |
  nix run .#scripts.parse-blocked-by-policy
```

This yields a long list of print statements issued during policy evaluation that allow tracing the execution.

<details>
<summary>It might look something like this:</summary>

```data
agent_policy:59:  CreateContainerRequest: i_oci.Hooks = null
agent_policy:62:  CreateContainerRequest: i_oci.Linux.Seccomp = null
agent_policy:66:  ======== CreateContainerRequest: trying next policy container
agent_policy:70:  CreateContainerRequest: p_pidns = false i_pidns = false
agent_policy:75:  CreateContainerRequest: p Version = 1.1.0-rc.1 i Version = 1.1.0-rc.1
agent_policy:78:  CreateContainerRequest: p Readonly = true i Readonly = true
agent_policy:96:  allow_anno 1: start
agent_policy:103:  allow_anno 2: p Annotations = {"io.katacontainers.pkg.oci.bundle_path":"/run/containerd/io.containerd.runtime.v2.task/k8s.io/$(bundle-id)","io.katacontainers.pkg.oci.container_type":"pod_sandbox","io.kubernetes.cri.container-type":"sandbox","io.kubernetes.cri.sandbox-id":"^[a-z0-9]{64}$","io.kubernetes.cri.sandbox-log-directory":"^/var/log/pods/$(sandbox-namespace)_$(sandbox-name)_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","io.kubernetes.cri.sandbox-namespace":"default","nerdctl/network-namespace":"^/var/run/netns/cni-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"}
agent_policy:104:  allow_anno 2: i Annotations = {"io.katacontainers.pkg.oci.bundle_path":"/run/containerd/io.containerd.runtime.v2.task/k8s.io/9aeb4dfa752e8607354029e3a22f0ddbb6f0aee44898ab4054ea8d50c595abcb","io.katacontainers.pkg.oci.container_type":"pod_sandbox","io.kubernetes.cri.container-type":"sandbox","io.kubernetes.cri.sandbox-cpu-period":"100000","io.kubernetes.cri.sandbox-cpu-quota":"0","io.kubernetes.cri.sandbox-cpu-shares":"2","io.kubernetes.cri.sandbox-id":"9aeb4dfa752e8607354029e3a22f0ddbb6f0aee44898ab4054ea8d50c595abcb","io.kubernetes.cri.sandbox-log-directory":"/var/log/pods/default_mongodb-74d9c7d6f5-b6tvr_268c2a50-6bf6-443f-8c65-59ebefabd52d","io.kubernetes.cri.sandbox-memory":"0","io.kubernetes.cri.sandbox-name":"mongodb-74d9c7d6f5-b6tvr","io.kubernetes.cri.sandbox-namespace":"default","io.kubernetes.cri.sandbox-uid":"268c2a50-6bf6-443f-8c65-59ebefabd52d","nerdctl/network-namespace":"/var/run/netns/cni-bedfd802-2831-03e4-2dfa-58569cfd2cfa"}
agent_policy:107:  allow_anno 2: i keys = ["io.katacontainers.pkg.oci.bundle_path","io.katacontainers.pkg.oci.container_type","io.kubernetes.cri.container-type","io.kubernetes.cri.sandbox-cpu-period","io.kubernetes.cri.sandbox-cpu-quota","io.kubernetes.cri.sandbox-cpu-shares","io.kubernetes.cri.sandbox-id","io.kubernetes.cri.sandbox-log-directory","io.kubernetes.cri.sandbox-memory","io.kubernetes.cri.sandbox-name","io.kubernetes.cri.sandbox-namespace","io.kubernetes.cri.sandbox-uid","nerdctl/network-namespace"]
agent_policy:117:  allow_anno_key 1: i key = io.katacontainers.pkg.oci.bundle_path
agent_policy:124:  allow_anno_key 2: i key = io.katacontainers.pkg.oci.bundle_path
agent_policy:129:  allow_anno_key 2: true
[...]
agent_policy:1116:  match_caps 3: start
agent_policy:1092:  allow_caps: policy Permitted = ["$(default_caps)"]
agent_policy:1093:  allow_caps: input Permitted = ["CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_FSETID","CAP_FOWNER","CAP_MKNOD","CAP_NET_RAW","CAP_SETGID","CAP_SETUID","CAP_SETFCAP","CAP_SETPCAP","CAP_NET_BIND_SERVICE","CAP_SYS_CHROOT","CAP_KILL","CAP_AUDIT_WRITE"]
agent_policy:1098:  match_caps 1: start
agent_policy:1105:  match_caps 2: start
agent_policy:1110:  match_caps 2: default_caps = ["CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_FSETID","CAP_FOWNER","CAP_MKNOD","CAP_NET_RAW","CAP_SETGID","CAP_SETUID","CAP_SETFCAP","CAP_SETPCAP","CAP_NET_BIND_SERVICE","CAP_SYS_CHROOT","CAP_KILL","CAP_AUDIT_WRITE"]
agent_policy:1113:  match_caps 2: true
agent_policy:1116:  match_caps 3: start
agent_policy:509:  allow_user: input uid = 101 policy uid = 65535
agent_policy:149:  allow_by_anno 2: start
agent_policy:66:  ======== CreateContainerRequest: trying next policy container
agent_policy:70:  CreateContainerRequest: p_pidns = false i_pidns = false
agent_policy:75:  CreateContainerRequest: p Version = 1.1.0-rc.1 i Version = 1.1.0-rc.1
agent_policy:78:  CreateContainerRequest: p Readonly = false i Readonly = true": unknown
```

</details>

The messages contain line numbers that allow following the execution in the policy source, which can be rendered with

```sh
cat deployment.yaml | nix run .#scripts.extract-policies
```

At first, we see that there are two `CreateContainerRequest: trying next policy container` messages.
For every `CreateContainerRequest`, the policy checks the input against all of the containers generated by genpolicy.
Usually, only one of the containers has meaningful overlap with the input and is thus likely to be the intended match.
In the log above this is the pause container, identified by the annotation `"io.kubernetes.cri.container-type":"sandbox"`.
For regular containers, their name is present in the annotation `io.kubernetes.cri.container-name`.

The messages usually follow this structure:

```data
<function> <id>: start
<function> <id>: input_data = value1 policy_data = value1
<function> <id>: true
```

The ID is necessary to tell rules with multiple definitions apart.
If a rule can't be satisfied, the corresponding `... true` statement is missing.
A good strategy for finding the failing rule is to go back to the last `... true` line and look at what follows immediately after.
For the example above, that would be

```data
agent_policy:1113:  match_caps 2: true
agent_policy:1116:  match_caps 3: start
agent_policy:509:  allow_user: input uid = 101 policy uid = 65535
agent_policy:149:  allow_by_anno 2: start
```

First, we look at `match_caps 3`.
It's unclear why this was even started, given that `match_caps 2` succeeded and returned to `allow_process`.
Anyway, the next rule to evaluate is `allow_user`, which has an obvious mismatch which turns out to be the root cause.
`allow_by_anno 2` is then tried as a last resort, but that fails early due to a missing annotation.

### Offline testing

An alternative strategy for testing policies is to launch a pod and collect the `CreateContainerRequest` observed by the agent.
See the [serial console](../serial-console.md) doc for instructions on how to do that.
Assuming the request encoded in `request.json` (in Rust's serialization!) and the generated policy in `policy.rego`, execute OPA with

```sh
opa eval  -i request.json -d policy.rego 'data.agent_policy.CreateContainerRequest'
```

## Problems with Generated Policy

Notice that the policy is generated from Kubernetes resource specs, but is applied to, say, `CreateContainerRequest` protobuf resource.
The following problem categories emerge from this design decision:

* Policy evaluation on API requests can't prevent events from *not* happening.
* Underspecified mapping from Kubernetes objects to OCI Runtime requests causes ambiguity.
* Configuration that can't be deterministically decided leaves sharp edges.

### Absence of Required Events

Today's policy evaluation can't verify the order of containers, or even their presence.
This is particularly damaging for init containers that maintain security invariants.

Fixing this would require a stateful policy evaluation that takes previous requests into account.
However, verifying the presence of non-init containers isn't feasible with this approach, but also less security critical.

Also affected by this are pod lifecycle hooks and probes.

### Ambiguity

The mapping from pod spec to OCI spec isn't specified, and the exact outcome strongly depends on the CRI.
For example, the CRI might set additional environment variables or mount points, or the Kubelet adds a `resolv.conf` mount.
The pause container used by CRIs is also a good example of an unspecified addition that needs to manifest in policy.

On the other hand, the policy needs to be explicit about what's allowed into the TEE, because many of the underspecified things can pose security risk - think `LD_PRELOAD` or mounting over `/bin/sh`.
Thus, the `genpolicy` tool needs to reproduce inner logic of the Kubelet and the target CRI to allow exactly what they're going to add to the spec.
This is primarily an engineering issue that makes CoCo difficult to port, but it also makes generated policies more obscure.

### Sharp Edges

Some parts of the container environment can't be checked by policy.
This puts the onus on the application to not trust these parts, deteriorating the lift-and-shift experience.
Examples include:

* Environment variables with dynamic information (such as provided by `PodSpec.enableServiceLinks`)
* DNS *configuration* (DNS *servers* can not be trusted anyway, which is a good reason to scrutinize DNS config)
* Other little things like downward API, generated names

## Open Questions

* Can DNS config from PodSpec be verified by policy at all?
* How do mounted ConfigMaps/Secrets behave in Kata?
  <https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically>
