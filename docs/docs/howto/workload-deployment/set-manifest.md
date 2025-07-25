# Set manifest

Setting the manifest enables the Contrast Coordinator to verify the deployment.

## Applicability

This step is mandatory for all Contrast deployments. Workloads won't start until
a valid manifest has been configured.

## Prerequisites

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Add Coordinator to resources](./add-coordinator.md)
5. [Prepare deployment files](./deployment-file-preparation.md)
6. [Configure TLS (optional)](./TLS-configuration.md)
7. [Enable GPU support (optional)](./GPU-configuration.md)
8. [Generate annotations and manifest](./generate-annotations.md)
9. [Deploy application](./deploy-application.md)

## How-to

### Connect to the Contrast Coordinator

The released Coordinator resource includes a LoadBalancer definition we can use.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

:::info[Port-forwarding of Confidential Containers]

`kubectl port-forward` uses a Container Runtime Interface (CRI) method that
isn't supported by the Kata shim. If you can't use a public load balancer, you
can deploy a port-forwarding pod to relay traffic to a Contrast pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: port-forwarder-coordinator
spec:
  containers:
    - name: port-forwarder
      image: alpine/socat
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

Upstream tracking issue:
https://github.com/kata-containers/kata-containers/issues/1693.

:::

### Set manifest

Attest the Coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

This will use the reference values from the manifest file to attest the
Coordinator. After this step, the Coordinator will start issuing TLS
certificates to the workloads. The init container will fetch a certificate for
the workload and the workload is started.
