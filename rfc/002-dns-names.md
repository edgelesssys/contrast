# RFC 002: DNS Names

Confidential pods should contain an environment value that contains a list
of additional DNS names the pod should receive.

## The Problem

We currently make use of certificates to distinguish between attested pods that
are part of one mesh and unattested pods. The coordinator who singes the
certificate for the workloads, does not bind the certificate to the identity of
the pod.

Since all certificates are wildcard certificates for any domain, every attested
pod can impersonate any workload inside its mesh.

For one, this can lead to confusion when we mention to users that we create
a "service mesh" between workloads as other service mesh implementations have a
strong sense of identity.

Moreover, while generally the attested workloads are seen as trusted and secure,
no sense of identity could make lateral movement through the confidential parts
of the cluster easier.

Lastly, identification of endpoints, debuggability and logging might be improved
through more information in the leaf certificates.

## Solution

Add the environment variable `EDG_DNS_NAMES` to every pod definition. The value
should be a comma separated list of dns names.

If unset, the fallback value of `*` will be used for usability.

The environment value is read by the initializer init container and used as
argument for the `NewMeshCert()` call.

Note that `NewMeshCert()` does not create the cert on the application layer but
only retrieves the cert created on layer 4.
In order to pass additional arguments from the initializer side, we'd need to
shift the certificate creation and therefore the attestation to the application
layer.

## Alternatives

Since using a different aTLS flow might not be desireable, we can consider the
following alternative:

Let the manifest set by the user inside the coordinator contain a mapping from
workload policy hash to DNS names. When creating the manifest we map each
workload to `*`. This can then be changed later by the user.

As an improvement we can add a annotation similar as described above containing
a comma separated list of DNS names. This is then used as an override for the
DNS names when generating the manifest.
