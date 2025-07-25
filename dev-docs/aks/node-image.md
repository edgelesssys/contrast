# AKS node image release flow

Most relevant Microsoft forks for us:

- https://github.com/microsoft/kata-containers
- https://github.com/microsoft/cloud-hypervisor
- https://github.com/microsoft/confidential-containers-containerd

## Release flow

1. [Release of kata-containers fork](https://github.com/microsoft/kata-containers/releases):
   [3.2.0.azl2](https://github.com/microsoft/kata-containers/releases/tag/3.2.0.azl2)
   and matching genpolicy version
2. [PR updates the SPEC files in `azurelinux`](https://github.com/microsoft/azurelinux/pull/9261).
   This also where cloud-hypervisor, and the utility VM kernel are updated.
3. [Release of `azurelinux`](https://github.com/microsoft/azurelinux/releases):
   [2.0.20240609](https://github.com/microsoft/azurelinux/releases/tag/2.0.20240609-2.0)
4. Release notes are added to
   [AKSCBLMarinerV2/gen2kata](https://github.com/Azure/AgentBaker/blame/master/vhdbuilder/release-notes/AKSCBLMarinerV2/gen2kata/):
   [202406.19.0](https://github.com/Azure/AgentBaker/blob/master/vhdbuilder/release-notes/AKSCBLMarinerV2/gen2kata/202406.19.0.txt).
   These include the
   [`azurelinux` release version](https://github.com/Azure/AgentBaker/blame/master/vhdbuilder/release-notes/AKSCBLMarinerV2/gen2kata/202406.19.0.txt#L696),
   as well as a
   [list that includes the relevant kata packages](https://github.com/Azure/AgentBaker/blame/master/vhdbuilder/release-notes/AKSCBLMarinerV2/gen2kata/202406.19.0.txt#L655-L667).
5. [Release of AKS](https://github.com/Azure/AKS/releases) mentions the Azure
   Linux image version in the release notes:
   [v20240627](https://github.com/Azure/AKS/releases/tag/2024-06-27) lists
   `AzureLinux-202406.19.0`
6. Rollout can be tracked on the
   [AKS release status page](https://releases.aks.azure.com/)
