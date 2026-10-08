# Connect to the Contrast Coordinator

This step describes how to connect to a running Contrast Coordinator.

## Applicability

This step is necessary for interacting with the Coordinator, which we'll need to be doing in some of the following steps.
In particular, a way to communicate with the Coordinator is required for [setting the manifest](./set-manifest.md) and for [verifying the deployment](./deployment-verification.md), two essential steps when working with Contrast.

## Prerequisites

1. [Set up cluster](../cluster-setup/bare-metal.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)
5. [Prepare deployment files](./deployment-file-preparation.md)
6. [Configure TLS (optional)](./TLS-configuration.md)
7. [Enable GPU support (optional)](./GPU-configuration.md)
8. [Generate annotations and manifest](./generate-annotations.md)
9. [Deploy application](./deploy-application.md)

## How-to

The released Coordinator resource includes a `LoadBalancer` definition.
Communication with the Coordinator can be handled via the ingress IP of the load balancer.
Store this IP in an environment variable for use in the following steps:

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

### Port forwarding

If you can't use a public load balancer, you can deploy a port-forwarding pod to relay traffic to a Contrast pod instead:

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

This is currently required because `kubectl port-forward` uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.
Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.
