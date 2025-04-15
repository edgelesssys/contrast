# Deploy Contrast Coordinator

This step adds an additional service to you cluster. The Coordinator takes care of verifying your deployment.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisite

1. [Set up cluster](../cluster-setup/aks.md)
2. [Install CLI](../install-cli.md)
3. [Deploy the Contrast runtime](./runtime-deployment.md)
4. [Prepare deployment files](./deployment-file-preparation.md)
5. [Configure TLS (optional)](./TLS-configuration.md)
6. [Enable GPU support (optional)](./GPU-configuration.md)
7. [Generate annotations and manifest](./generate-annotations.md)
8. [Deploy application](./deploy-application.md)

## How-to

### Download the Contrast Coordinator resource

You can choose whether to download and deploy the Coordinator manifest separately after deploying your application, or include it alongside your application resources and apply everything at once.
Note that your workloads won't run until a valid manifest has been set.

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a LoadBalancer service. Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir deployment
```

### Deploy the Coordinator

Deploy the Coordinator resource by applying its resource definition:

```sh
kubectl apply -f deployment/coordinator.yml

```

### Connect to the Contrast Coordinator

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

Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.

:::

### Set the manifest

Attest the Coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

This will use the reference values from the manifest file to attest the Coordinator.
After this step, the Coordinator will start issuing TLS certificates to the workloads. The init container
will fetch a certificate for the workload and the workload is started.
