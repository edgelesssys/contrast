# RFC 008: Platform Support

## Objective

Find an in-code representation for Contrast platforms that enables conditionals
based on platform features, rather than full platform definitions.

## Problem statement

Contrast currently distinguishes between platforms based on their distinct
features. Taking the `AKS-CLH-SNP` platform as an example, this serialized
platform string conveys the following information:

- Kubernetes engine: `AKS`
  - (Implicit: Kata upstream / Microsoft fork)
- Hypervisor: `CLH`
- Hardware platform: `SNP`

The user picks such a platform through its serialized string (for example
`AKS-CLH-SNP`), and the code then implements actions based upon that, such as
when writing platform-specific Kata configuration values in the
[node-installer].

To perform these actions in the code, conditionals based on the platform are
necessary.

Currently, platforms are represented as an int alias type:

```go
type Platform int
```

The individual platforms thus are represented by instantiations of said type:

```go
AKSCloudHypervisorSNP Platform = 1
```

Conditionals thus need to work on full platforms, making them look like:

```go
// TDX platforms only
if platform == platforms.K3sQEMUTDX || platform == platforms.RKE2QEMUTDX {
  // ..
}
```

This isn't only cumbersome to write, but also easy to make errors in. For
example when forgetting to match against a certain platform.

[node-installer]: https://github.com/edgelesssys/contrast/blob/3f39682ea9a383b2557923e257bd065e461b8ee6/nodeinstaller/internal/constants/constants.go#L48

## Proposal

### Structured Representation

Throughout the implementation code, platforms shouldn't be represented as
_individual combinations_, but rather through a generic, structured type that
contains information about their _features_.

So `AKS-CLH-SNP` could become something like this:

```go
type Platform struct {
    ContainerdConfigPath  string
    UnitsToRestart        []string
    Snapshotter           Snapshotter
    GPUSupport            bool
    Debug                 bool
    Hypervisor            Hypervisor
    HardwarePlatform      HardwarePlatform
}

// AKS-CLH-SNP
foo := Platform{
    ContainerdConfigPath:   "/etc/containerd/config.tomL",
    UnitsToRestart:         []string{"containerd.service"},
    GPUSupport:             false,
    Debug:                  false,
    Hypervisor:             CloudHypervisor,
    HardwarePlatform:       SNP,
}
```

This would allow the implementation code (for example the aforementioned switch
for [Kata configuration]) to be reduced to only depend on the information bits
it requires.

For example, this:

```go
case platforms.MetalQEMUTDX, platforms.K3sQEMUTDX, platforms.RKE2QEMUTDX:
```

Could become this:

```go
case TDX
```

This would also enable easy validation of user error, as the individual types
can be checked for validity at parsing time, making invalid states
unrepresentable [^1].

[Kata configuration]: https://github.com/edgelesssys/contrast/blob/3f39682ea9a383b2557923e257bd065e461b8ee6/nodeinstaller/internal/constants/constants.go#L48

[^1]: https://geeklaunch.io/blog/make-invalid-states-unrepresentable/
