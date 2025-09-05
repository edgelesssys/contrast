# Deployment verification

This step verifies that the application has been deployed correctly.

## Applicability

An application user can use this step to verify the deployment before interacting with the application.

## Prerequisites

1. A running Contrast deployment.
2. [Install CLI](../install-cli.md)

## How-to

This page explains how a user can connect to the Coordinator and verify the application's integrity.

### Connect to the Coordinator

For the next steps, we will need to connect to the Coordinator. The released Coordinator resource
includes a LoadBalancer definition we can use.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

:::info[Port-forwarding of Confidential Containers]

`kubectl port-forward` uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.
If you can't use a public load balancer, you can deploy a port-forwarding pod to relay traffic to a Contrast pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: port-forwarder-coordinator
spec:
  containers:
    - name: port-forwarder
      image: "alpine/socat@sha256:..."
      args:
        - -d
        - TCP-LISTEN:1313,fork
        - TCP:coordinator:1313
      resources:
        requests:
          memory: 50Mi
        limits:
          memory: 50Mi
```

Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.

:::

### Verify deployment

An end user (data owner) can verify the Contrast deployment using the `verify` command.

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using the reference values from the given manifest file. It will then write the
service mesh root certificate and the history of manifests into the `verify/` directory. In addition, the policies
referenced in the active manifest are also written to the directory. The verification will fail if the active
manifest at the Coordinator doesn't match the manifest passed to the CLI.
