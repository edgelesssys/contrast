---
slug: /faq
id: faq
---

# Frequently Asked Questions

Below, you'll find answers to the questions most frequently asked about Contrast.
If your question is answered neither in the documentation nor in this FAQ, please reach out via [GitHub discussions](https://github.com/edgelesssys/contrast/discussions/new/choose).

## Do network policies work with Contrast?

Since the CNI doesn't have visibility into pod-level traffic, standard Kubernetes `NetworkPolicies` don't work out of the box.

## Which Container Network Interfaces are compatible with Contrast?

Contrast is supported on a variety of Container Network Interface (CNI) implementations.
Please see the docs section on [networking](./howto/hardening.md#networking) for more details.
