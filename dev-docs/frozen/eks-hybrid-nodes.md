# AWS EKS hybrid nodes

EKS allows the customer to join nodes they themselves manage. See [this table](https://aws.amazon.com/blogs/aws/use-your-on-premises-infrastructure-in-amazon-eks-clusters-with-amazon-eks-hybrid-nodes/)
for what layer is managed by whom.

## Use cases
1. Customer already is using EKS and want to use Contrast. Note that the customer may not already own on-prem infrastructure.
1. Customer already has multiple on-prem bare-metal nodes that they want to control via a managed Kubernetes.

## What needs to be done

Their networking guide [https://docs.aws.amazon.com/eks/latest/userguide/hybrid-nodes-networking.html](https://docs.aws.amazon.com/eks/latest/userguide/hybrid-nodes-networking.html)
illustrates the VPC peering that's needed to communicate between the EKS cluster and the on-prem nodes.
Note that if we assume that the customer has on-prem server, they also should manage the access to their
on-prem network themselves. Therefore, they must implement the changes to connect their on-prem nodes to EKS.

Contrast needs to build on top of and support:
* the containerd version
* the containerd setting paths
* the given block storage (though we build on top of the general Kubernetes abstraction here)

## How a Contrast test setup could look like

There are two options to set this up _without_ having an enterprise on-prem datacenter:
1. Add single Hetzner bare-metal nodes via strongSwan
1. Peer an Azure VPC with SNP-nested VMs

There are trade-offs for both:
The Hetzner way is more effort to setup the custom strongSwan and is off the golden path set
by the AWS docs. The benefit is that the nodes added are bare-metal nodes and should behave
similar to customer bare-metal nodes.

The Azure way is more aligned with the AWS docs regarding the networking, though
the nodes are already virtualized and the are using cloud-hypervisor instead of
qemu, which will differ from the customers' bare-metal nodes.

## Alternatives

A customer might not want to use Contrast on hybrid nodes because:
1. AWS might not be their cloud of choice.
1. They're using a self-managed Kubernetes (for example OpenShift) for on-prem deployments.
1. If they have to use on-prem bare-metal nodes, they might be fine not using CC.

Moreover, if AWS enables and supports SNP VMs on their `.metal` instances, which
are natively supported in EKS, we don't need hybrid nodes to support Contrast on
EKS. Note that the second use-case remains, but given the alternatives, is a weaker one.

## Summary

There are multiple alternatives to using Contrast on bare-metal with EKS hybrid nodes.
While feature itself might be small, it introduces a new platform or at least testing target
where we've to manage a lot of components in the CI.
