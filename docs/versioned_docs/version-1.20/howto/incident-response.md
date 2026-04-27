# Incident response

This document describes how to respond to security incidents affecting Contrast or your workloads.

## How-to

Incident response, by nature, is often a manual process requiring expertise in several domains.
In here, we're presenting a generic workflow with considerations and recommendations for specific incidents.
This guide is meant to inform your own incident response process, not prescribe it.

### Security advisories

Vulnerabilities for Contrast are published as Github Security Advisories (GHSA) on the [Contrast security] landing page.
They usually describe which components are affected and what remediation steps are in order.
Any steps recommended here should be double-checked against the advisory, if applicable.

[Contrast security]: https://github.com/edgelesssys/contrast/security/advisories

### Identify the scope

First, you will need to understand what parts of your stack are affected by a vulnerability or compromise, and to what extent.
Regarding Contrast, there are three scenarios worth disambiguating.

1. A workload is potentially vulnerable, but you can rule out active exploitation.
2. A workload is assumed to be compromised.
3. The Contrast Coordinator is assumed to be compromised.

### Remedy the situation

The necessary containment and remediation steps for the scopes identified above are incremental, meaning that the remediation for the most severe compromise should include all remediation steps for the less severe ones.

#### Vulnerable workload

If your workload is potentially vulnerable but you can rule out a compromise, the situation isn't too different from a software update.
Simply follow the instructions in the [manifest update](manifest-update.md) guide after upgrading your resource definitions to a secure version.
Additionally, inform any relying parties that use Coordinator CA certificates as trust anchors of the changed manifest and the new, trustworthy CA certificate.

#### Compromised workload

A workload needs to be assumed compromised if either the main containers or the sidecar containers provided by Contrast are vulnerable and exploitation can't be ruled out.
In that case, all secrets that can be accessed by this workload need to be assumed compromised, too.
This affects the [mesh certificate]  key, the [workload secret] and any secrets derived from it (notably, encryption keys for persistent volumes).

The mesh certificate key will be invalidated by the manifest update (see above), but the workload secret won't.
If preserving the Coordinator root secret isn't a concern, the simplest remediation is to set up a fresh Coordinator (see next section).
Otherwise, you need to at least ensure that the compromised workload secret isn't relied on in the future.
The process for assigning new workload secrets is described in the workload secrets documentation.
Be aware of the automatic workload secret ID assignment mechanism and make sure that no future workload receives the same ID as the compromised one.

[mesh certificate]: ../architecture/components/service-mesh.md
[workload secret]: ../architecture/secrets.md

#### Compromised Coordinator

In this scenario, you assume that the Contrast Coordinator has a vulnerability and you can't rule out that it's being exploited.
The Coordinator has access to all Contrast secrets, by design, which implies that all secrets need to be assumed compromised, too.

Since the existing secrets can't be trusted anymore, the only remediation is to create an entirely new Coordinator.
For that, first delete the existing Coordinator `StatefulSet`.
This should automatically clear the Coordinator's state in cluster `ConfigMaps`.
Then, apply the new Coordinator from the release containing the fix and set it up with a manifest.

Naturally, the new Coordinator shouldn't be configured with a manifest that would allow a vulnerable Coordinator.
