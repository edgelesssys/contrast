# Deployment verification

This step verifies that the application has been deployed correctly.

## Applicability

An application user can use this step to verify the deployment before interacting with the application.

## Prerequisites

1. A running Contrast deployment.
2. [Install CLI](../install-cli.md)

## How-to

This page explains how a user can connect to the Coordinator and verify the application's integrity.

An end user (data owner) can verify the Contrast deployment using the `verify` command.

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using the reference values from the given manifest file. It will then write the
service mesh root certificate and the history of manifests into the `verify/` directory. In addition, the policies
referenced in the active manifest are also written to the directory. The verification will fail if the active
manifest at the Coordinator doesn't match the manifest passed to the CLI.
