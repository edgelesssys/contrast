# RFC 008: Platform Support

## Objective

Implement support for diverse platforms in Contrast, ensuring that:

1. Users can mix-and-match features throughout all platforms supported by Contrast
2. Code with a platform dependency can be deduplicated
3. New platforms and platform features can be added with low complexity (that's `O(1)`, at best)

## Problem statement

Contrast currently distinguishes between platforms based on their distinct features.
Taking the `AKS-CLH-SNP` platform as an example, this serialized platform string conveys
the following information:

- Kubernetes engine: `AKS`
  - (Implicit: Kata upstream / Microsoft fork)
- Hypervisor: `CLH`
- Hardware platform: `SNP`

The user picks such a platform through its serialized string (for example `AKS-CLH-SNP`), and the
code then implements actions based upon that, such as when writing platform-specific Kata
configuration values in the [node-installer].

With new features being added to Contrast, such as GPU support, more information needs
to be contained in such a platform description.

This comes at the cost of growing complexity of conditional statements
depending on the platform throughout the implementation code, as the
current implementation doesn't distinguish between platform features
(for example which hypervisor is used) but only between full platform combinations.

[node-installer]: https://github.com/edgelesssys/contrast/blob/3f39682ea9a383b2557923e257bd065e461b8ee6/nodeinstaller/internal/constants/constants.go#L48

## Proposal

### Structured Representation

Throughout the implementation code, platforms shouldn't be represented as _individual combinations_,
but rather through a generic, structured type that contains information about their _features_.

So `AKS-CLH-SNP` could become something like this:

```go
type Platform struct {
    KubernetesDistribution KubernetesDistribution
    Hypervisor Hypervisor
    HardwarePlatform HardwarePlatform
}

// AKS-CLH-SNP
foo := Platform{
    KubernetesDistribution: AKS,
    Hypervisor: CloudHypervisor,
    HardwarePlatform: SNPMilan,
}
```

This would allow the implementation code (for example the aforementioned switch for [Kata configuration])
to be reduced to only depend on the information bits it requires.

For example, this:

```go
case platforms.MetalQEMUTDX, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
```

Could become this:

```go
case TDX
```

This would also enable easy validation of user error, as the individual types can be checked for
validity at parsing time, making invalid states unrepresentable [^1].

However, to get to such a structured format, the information needs to be passed to Contrast in another
structured format, such as a configuration file or another canonical serialized format, which should
be discussed in the following section.

[Kata configuration]: https://github.com/edgelesssys/contrast/blob/3f39682ea9a383b2557923e257bd065e461b8ee6/nodeinstaller/internal/constants/constants.go#L48

### Serialization

As platforms are passed throughout multiple (programming) languages, such as the `justfile` and Go code for
developers, but - in the most primitive way - from a user invoking the CLI in a shell to Go code, there must
be a serializable, machine-readable representation of a platform.

This could be a configuration file that's read by the CLI and other dependants.

This would reduce implementation complexity, as Contrast could rely on existing YAML / TOML / JSON / etc. parsing
libraries and no canonical string representation of a platform would need to be implemented.

Additionally, adding platform features would not lead to unreasonably large string representations, which might
be hard to read for humans at some point.

## Considerations

### Node-Installer Images

As it would be complex and costly to publish node-installer images for all supported platform
combinations, it might be better to ship a singular one-size-fits-all node installer with support
for all platforms. This could also be a foundation for heterogeneous deployments that utilize multiple
platforms (for example GPU- and Non-GPU machines).

To ensure a good developer experience, bandwidth used for transferring the node-installer images should
be kept to a minimum, which is contrary to the aforementioned idea, but could be counteracted with
minimizing the individual components in the node-installer, such as the node images.

[^1]: https://geeklaunch.io/blog/make-invalid-states-unrepresentable/
