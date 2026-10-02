# Why Contrast?

Contrast isn't the only software that runs containers in confidential VMs.
However, we believe that Contrast has some unique advantages that you should know about, which are listed in the following sections.

## Ready for production

Contrast is stable and secure software you can rely on in production.
The threat model is documented and we ensure the confidentiality and integrity of your containers.
Workloads automatically receive a restrictive policy, without leaving anything for you to figure out.
Contrast doesn't have a single-point-of-failure in the verification path and scales with your cluster.

## Lift and shift

Contrast is designed to support existing Kubernetes workloads without modifications.
You don't need to interact with remote attestation unless you want to, and you can even leave storage and network encryption to Contrast components.

## Reproducible from available source

Our software is developed in the open, without hidden components or gated enterprise features.
With access to the source code you can convince yourself of its security, and security researchers are invited to do so, too.
Our build system ensures outputs are reproducible, so you can verify binary artifacts yourself and don't need to trust our processes.

## Measurements included

Managing reference values for confidential computing can be quite cumbersome, in particular with diverse workloads.
Contrast manages all that complexity for you.
You don't need to use any third party tools or empirically derive appraisal policies.

## SaaS-ready

A Contrast deployment can be verified end-to-end, without requiring administrative access.
This allows offering software-as-a-service to users and excluding yourself from the trusted computing base.
Changes in software or configuration are measured and observable by your users.

## Plugs into standards

Contrast aims to comply with standards and established practice, in order to ease lifting a service into a confidential environment.
Contrast verification is X.509 based, making it compatible with most software stacks in existence.
Standard OCI images and registry authentication are supported out-of-the-box.
Adding a Vault to manage your secrets is straightforward and fully integrated into attestation.

## The team

Contrast was created by Edgeless Systems, the confidential computing pioneers from Germany.
We've been scaling confidential computing since 2020, with a team of cybersecurity and system engineering experts.
Contrast is our third product at the intersection of confidential computing and Kubernetes, and our designs draw from this vast experience.
