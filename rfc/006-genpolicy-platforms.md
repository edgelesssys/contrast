# RFC 006: `genpolicy` for diverse platforms

This RFC describes the current situation of `genpolicy` and proposes a strategy for making it work for all supported Contrast platforms.

## Problem Statement

Generated policies are a key building block of Confidential Containers and Contrast.
However, the only implementation that guarantees basic confidentiality levels is the fork maintained by Microsoft.
The upstream Kata policy is lacking security fixes that haven't been backported, and yet lacks a mechanism for ensuring image integrity.
On the other hand, some features that we contributed and would like to use are only available upstream with no ETA of adoption by Microsoft.

We're using the Microsoft fork successfully on AKS-CoCo, but it's tailored to that platform and unlikely to ever be portable.
Portability will become important for our bare-metal efforts, though.
Bare-metal clusters will diverge from the standardized AKS environment, requiring flexibility in policy generation.

## Parts of genpolicy

The workload policy mechanism can be divided into four parts:

* The tool that generates reference values from workload YAML (`genpolicy`).
* The settings file for the tool (`genpolicy-settings.json`).
* The policy rules that allow agent queries based on reference values (`rules.rego`).
* The policy engine that's queried in the agent.

These components are somewhat independent, and the following subsections describe how they fit together.

### The tool

The idea of the `genpolicy` tool is to take a set of Kubernetes workload descriptors and derive the Kata agent API requests that a legitimate runtime would issue to manage these workloads.
The output of the tool are reference values that can be referred to from rules (see below).

Implementing this idea is complicated, because there is no 1:1 correspondence between, say, a pod definition and the corresponding agent method calls.
The pod undergoes a few transformations in between API server, kubelet, containerd and Kata, which the tool needs to emulate to some extent.
More information on this problem is available in the [policy report](../dev-docs/coco/policy.md).

Guessing the transformations from Kubernetes to CRI is easy because it's somewhat stable.
The path from Kata runtime to the agent is well-known, provided that `genpolicy` and Kata runtime are build from the same version.
In between sits containerd, for which some assumptions need to be made - this is where the settings enter the picture, see below.

The container image integrity is verified by calculating `dm-verity` hashes of the layer tarballs and including these in the container reference values.
These hashes are only used in Microsoft's policy, though - upstream, the corresponding rules are dead code waiting to be replaced by another integrity checking mechanism.

### The settings

Settings are designed to model behavior that's dependent on the environment or specific to a certain cluster configuration.
For example, a newer containerd might inject `sysctl` rules that an older one didn't, or a cluster administrator might want to mount devices (like GPUs) with a device plugin.

### The rules

The rules consist of predicates that evaluate to a Boolean decision for all agent service methods.
Broadly generalizing, each rule compares the input request with reference values.
If the input request diverges from expectations, the output is `false` and the request is rejected.

The final workload policy is assembled by appending the reference values as a `policy_data` field to the rules file.

Assuming a relatively stable format of the reference values, the rules can be written independently of the other parts.
For example, if `genpolicy` computes both `dm-verity` hashes for tardev layers and includes the corresponding pinned images, rules can be written to accommodate both the tardev-snapshotter and any guest-pull mechanism.

### The agent

The functionality in the agent is very simple: an in-memory policy engine is equipped with the incoming policy, and subsequent agent requests are subject to policy validation. The agent starts from a default policy that's available in the UVM image.
When a container is started with the tardev-snapshotter, the agent also ensures that the layers are mounted with `dm-verity` protection.

## State of genpolicy

This is a snapshot of the features in Microsoft's fork that are yet to be implemented in upstream Kata: <https://gist.github.com/burgerdev/9330236c209f49c9d97e01f278b3a915>.
Setting aside the changes for Azure's CSI drivers, the two main categories are YAML support and hardening.
Both of these are desirable to backport, the difficulty stems from code layout changes between the repositories.

Contrast uses the Microsoft fork of `genpolicy` and its configuration, with a few patches on top.

The upstream policy has support for `dm-verity` calculation, but the corresponding rule is commented out.
It also uses an outdated and unmaintained version of `tarindex`.
Upstream will need to introduce support for other snapshotters (that is, Nydus) eventually.

## Proposal

Contrast shouldn't need to package different versions of the `genpolicy` tool or the CLI.
Instead, the upstream tool should be able to cover generation for all target platforms.
Contrast should bundle rules and settings for its target platforms, and invoke the tool accordingly.

How do we get there?

### Focus on upstream

The Kata `genpolicy` is already close to the desired state, and will get even more traction once the CoCo community needs policy enforcement.
We can accelerate progress by porting necessary changes ourselves.
Also, contributions from Contrast can ensure that the tool becomes flexible enough to support our use cases.

### Prepare to maintain patches

We would like to support a mix-and-match of snapshotters and target platforms, but the CoCo community consensus is to focus on one.
Thus, it's likely that Contrast will need patches, for example for an updated `tarindex` or even the entire `dm-verity` calculation.

### Develop our own rules and settings

Due to the nature of the settings file, Contrast can't rely on a single vendor to provide this.
On the other hand, we don't want to force our users to make modifications for our standard target platforms.
Thus, we develop settings for our known target platforms, validate them through CI and bundle them with the CLI.

The upstream rules should eventually be compatible with all non-tardev platforms.
This is a shared interest throughout the wider CoCo community, but we can accelerate it through contributions.

For the tardev snapshotter platforms (that is, AKS for now), we start with the Microsoft rule set.
The next step is to write a rule set and settings file compatible with upstream genpolicy reference values, targeted towards AKS CoCo.

## Alternatives considered

### Bundle both Microsoft's and Kata's tool

While this approach seems easiest on the surface, it's going to explode the size of the CLI binary, which is already significant.
Furthermore, it introduces complexity all over the codebase, from packaging to invocation.
It's also unlikely that we can use both tools as-are, so we would need to maintain two sets of patches.
Context-switching between the two tools might make writing policies even harder.

### Roll our own tool

This approach is tempting, because we could integrate it better into our packages and tailor it to our needs.
However, there's a non-negligible risk that the upstream approach changes incompatibly, forcing us to also carry patches for the other Kata components.
We would also not immediately benefit from upstream activity, in particular security reviews.
