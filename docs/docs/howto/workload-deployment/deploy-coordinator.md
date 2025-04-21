# Deploy Contrast coordinator

This step adds an additional service to you cluser. The coordinator takes care of verifying your deployment.

## Applicability

This step is mandatory for all Contrast deployments.

## Prerequisite

1. [Set up cluster](.)
2. [Deploy runtime](.)
3. [Prepare deployment files](.)
4. [Configure TLS (optional)](.)
5. [Enable GPU support (optional)](.)
6. [Generate annotations and manifest](.)
7. [Deploy application](.)

## How-to

### Download the Contrast coordinator resource

Download the Kubernetes resource of the Contrast coordinator, comprising a single replica deployment and a
LoadBalancer service. Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir deployment
```

### Deploy the coordinator

Deploy the coordinator resource by applying its resource definition:

```sh
kubectl apply -f deployment/coordinator.yml

```

### Connect to the Contrast coordinator

For the next steps, we will need to connect to the coordinator. The released coordinator resource
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

Attest the coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

This will use the reference values from the manifest file to attest the coordinator.
After this step, the coordinator will start issuing TLS certificates to the workloads. The init container
will fetch a certificate for the workload and the workload is started.
