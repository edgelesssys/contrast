# AKS Confidential Containers

The Contrast stack on AKS is based on the [AKS Confidential Containers preview].
Azure has support for
[nested virtualization](../frozen/aks-nested-virt-internals.md) in Hyper-V,
which allows spawning CVMs from AKS node VMs. The preview provides a special
node runtime, `KataCcIsolation`, equipped with Microsoft's fork of Kata
Containers.

[AKS Confidential Containers preview]: https://learn.microsoft.com/en-us/azure/aks/confidential-containers-overview

## Sources

The forked code is available at <https://github.com/microsoft/kata-containers>.
The stated goal of Microsoft is to keep the fork somewhat close to upstream, and
Microsoft engineers regularly backport selected commits from upstream Kata. As
of 2024-06-26, the codebases have diverged, in particular for the `genpolicy`
tool, so that most patches don't apply cleanly.

The `KataCcIsolation` nodes support the runtime class `kata-cc-isolation`, and
can be configured to use [debug images](../serial-console).

## Compatibility

According to sources at Microsoft, the `KataCcIsolation` nodes are compatible
with the most recent version of the `az confcom` extension. This extension is
available at <https://github.com/Azure/azure-cli-extensions>. Judging from the
[version selection algorithm], the currently supported version is the latest
GitHub release containing `genpolicy`.

[version selection algorithm]: https://github.com/Azure/azure-cli-extensions/blob/417b468/src/confcom/azext_confcom/kata_proxy.py#L39-L73
