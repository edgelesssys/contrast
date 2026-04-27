# Deployment verification

This step verifies that the application has been deployed correctly.

## Applicability

An application user can use this step to verify the deployment before interacting with the application.

## Prerequisites

1. A running Contrast deployment.
2. [Install CLI](../install-cli.md)

## How-to

This page explains how a user can connect to the Coordinator and verify the application's integrity.
You can start the verification at any of the following subsections with the outputs of the preceding sections, provided that you received these outputs from a trustworthy source.

All steps in this guide must be executed on a trustworthy system under your control.
This applies to all binaries used in the process, too, in particular the Contrast CLI.

<!-- TODO(burgerdev): document how to build trust in the Contrast CLI. -->

### Generate an expected manifest

This step assumes that you're verifying a Contrast deployment owned by a third party.
If you're the workload owner and generated the manifest, you can proceed to the next section.

First, obtain the Kubernetes resource definitions that the workload owner used to generate a manifest.
If the workload owner modified the manifest after generating it, you will need to apply the same modifications.
This applies to changes of the `genpolicy-settings.json` and the `rules.rego`, too (see the [policy documentation](../../architecture/components/policies.md) for details on these files).

Now review all pod definitions with a `contrast-cc` runtime class name and supplemental resources like `ConfigMap`s.
Depending on the requirements, this review could potentially go very deep.
At the very least, you should ensure that the images correspond to expected released versions and that the `arg` and `env` fields look plausible.

Finally, [generate the expected manifest](generate-annotations.md) as you would for your own application, applying the same changes as the workload owner.
Make sure to use the same Contrast version as the workload owner.
If the workload owner published their manifest somewhere, compare it with the one you generated.
It should be byte-for-byte identical.

:::note

While it's theoretically possible to audit a Coordinator with just the manifest and initdata documents, we strongly advise against that.
It's easy to miss subtle differences in the generated policies, so reproducing the manifest is a lot easier than auditing it.

:::

### Verify the Coordinator manifest

This step assumes that you have a trusted `manifest.json` in the current working directory.
Run the `verify` subcommand to check that:

1. The Coordinator runs in a TEE that's allowed by the manifest's reference values.
2. The Coordinator runs the expected software (one of the policies with `Role: Coordinator` matches).
3. The current manifest is exactly equal to the manifest provided to the CLI.

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI writes the root CA certificate, the mesh CA certificate and the history of manifests into the `verify/` directory.
In addition, the initdata documents referenced in the active manifest are also written to the directory.
The verification will fail if the active manifest at the Coordinator doesn't match the manifest passed to the CLI.
Consult the [manifest reference](../../architecture/components/manifest.md) to understand what aspects of the workload are evaluated.

### Verify the application

In this step, you verify that your application successfully attested to the Coordinator and received its [mesh certificate](../../architecture/components/service-mesh.md#public-key-infrastructure).
You will need the `mesh-ca.pem` certificate for this, which was written to the `verify/` directory in the previous step.

If your application exposes a TLS server on a public port that uses its mesh certificate, you can verify that certificate locally with the following `openssl` command.

```sh
openssl s_client -verify_return_error -x509_strict  -connect "$POD_IP:$PORT" -CAfile mesh-ca.pem </dev/null
```

A successful response includes the message `Verify return code: 0 (ok)` at the end, while a verification failure is indicated by a non-zero exit code.
