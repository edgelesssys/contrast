# Workload deployment

The following instructions will guide you through the process of making an existing Kubernetes deployment
confidential and deploying it together with Contrast.

A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/cluster-setup.md) on how to set it up.

## Deploy the Contrast Coordinator

Install the latest Contrast Coordinator release, comprising a single replica deployment and a
LoadBalancer service, into your cluster.

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml
```

## Prepare your Kubernetes resources

Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files
unchanged, you can copy the files into a separate local directory.
You can also generate files from a Helm chart or from a Kustomization.

<Tabs groupId="yaml-source">
<TabItem value="kustomize" label="kustomize">

```sh
mkdir resources
kustomize build $MY_RESOURCE_DIR > resources/all.yml
```

</TabItem>
<TabItem value="helm" label="helm">

```sh
mkdir resources
helm template $RELEASE_NAME $CHART_NAME > resources/all.yml
```

</TabItem>
<TabItem value="copy" label="copy">

```sh
cp -R $MY_RESOURCE_DIR resources/
```

</TabItem>
</Tabs>

To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,
add `runtimeClassName: kata-cc-isolation` to the pod spec (pod definition or template).
In addition, add the Contrast Initializer as `initContainers` to these workloads and configure the
workload to use the certificates written to a `volumeMount` named `tls-certs`.

```yaml
spec: # v1.PodSpec
  runtimeClassName: kata-cc-isolation
  initContainers:
  - name: initializer
    image: "ghcr.io/edgelesssys/contrast/initializer:latest"
    env:
    - name: COORDINATOR_HOST
      value: coordinator
    volumeMounts:
    - name: tls-certs
      mountPath: /tls-config
  volumes:
  - name: tls-certs
    emptyDir: {}
```

## Generate policy annotations and manifest

Run the `generate` command to generate the execution policies and add them as annotations to your
deployment files. A `manifest.json` with the reference values of your deployment will be created.

```sh
contrast generate resources/
```

## Apply the resources

Apply the resources to the cluster. Your workloads will block in the initialization phase until a
manifest is set at the Coordinator.

```sh
kubectl apply -f resources/
```

## Connect to the Contrast Coordinator

For the next steps, we will need to connect to the Coordinator. The released Coordinator resource
includes a LoadBalancer definition we can use.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

:::info[Port-forwarding of Confidential Containers]

`kubectl port-forward` uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.
If you can't use a public load balancer, you can deploy a [port-forwarder](https://github.com/edgelesssys/contrast/blob/main/deployments/emojivoto/portforwarder.yml).
The port-forwarder relays traffic from a CoCo pod and can be accessed via `kubectl port-forward`.

Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.

:::

## Set the manifest

Attest the Coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

After this step, the Coordinator will start issuing TLS certs to the workloads. The init container
will fetch a certificate for the workload and the workload is started.

## Verify the Coordinator

An end user (data owner) can verify the Contrast deployment using the `verify` command.

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using embedded reference values. The CLI will write the service mesh
root certificate and the history of manifests into the `verify/` directory. In addition, the policies referenced
in the manifest are also written to the directory.

## Communicate with workloads

You can securely connect to the workloads using the Coordinator's `mesh-root.pem` as a trusted CA certificate.
First, expose the service on a public IP address via a LoadBalancer service:

```sh
kubectl patch svc ${MY_SERVICE} -p '{"spec": {"type": "LoadBalancer"}}'
timeout 30s bash -c 'until kubectl get service/${MY_SERVICE} --output=jsonpath='{.status.loadBalancer}' | grep "ingress"; do sleep 2 ; done'
lbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo $lbip
```

:::info[Subject alternative names and LoadBalancer IP]

By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed
via load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.
Validation fails since the certificate contains no IP entries as a subject alternative name (SAN).
For example, a connection attempt using the curl and the mesh root certificate with throw the following error:

```sh
$ curl --cacert ./verify/mesh-root.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

:::

Using `openssl`, the certificate of the service can be validated with the `mesh-root.pem`:

```sh
openssl s_client -CAfile verify/mesh-root.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```
