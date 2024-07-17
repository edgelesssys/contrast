# Workload deployment

The following instructions will guide you through the process of making an existing Kubernetes deployment
confidential and deploying it together with Contrast.

A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/cluster-setup.md) on how to set it up.

## Deploy the Contrast runtime

Contrast depends on a [custom Kubernetes `RuntimeClass` (`contrast-cc`)](./components/runtime.md),
which needs to be installed in the cluster prior to the Coordinator or any confidential workloads.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v0.8.0/runtime.yml
```

## Deploy the Contrast Coordinator

Install the latest Contrast Coordinator release, comprising a single replica deployment and a
LoadBalancer service, into your cluster.

```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v0.8.0/coordinator.yml
```

## Prepare your Kubernetes resources

Your Kubernetes resources need some modifications to run as Confidential Containers.
This section guides you through the process and outlines the necessary changes.

### RuntimeClass

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
add `runtimeClassName: contrast-cc` to the pod spec (pod definition or template).
This is a placeholder name that will be replaced by a versioned `runtimeClassName` when generating policies.

```yaml
spec: # v1.PodSpec
  runtimeClassName: contrast-cc
```

### Handling TLS

In the initialization process, the `contrast-tls-certs` shared volume is populated with X.509 certificates for your workload.
These certificates are used by the [Contrast Service Mesh](components/service-mesh.md), but can also be used by your application directly.
The following tab group explains the setup for both scenarios.

<Tabs groupId="tls">
<TabItem value="mesh" label="Drop-in service mesh">

Contrast can be configured to handle TLS in a sidecar container.
This is useful for workloads that are hard to configure with custom certificates, like Java applications.

Configuration of the sidecar depends heavily on the application.
The following example is for an application with these properties:

* The container has a main application at TCP port 8001, which should be TLS-wrapped and doesn't require client authentication.
* The container has a metrics endpoint at TCP port 8080, which should be accessible in plain text.
* All other endpoints require client authentication.
* The app connects to a Kubernetes service `backend.default:4001`, which requires client authentication.

Add the following annotations to your workload:

```yaml
metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...
  annotations:
    contrast.edgeless.systems/servicemesh-ingress: "main#8001#false##metrics#8080#true"
    contrast.edgeless.systems/servicemesh-egress: "backend#127.0.0.2:4001#backend.default:4001"
```

During the `generate` step, this configuration will be translated into a Service Mesh sidecar container which handles TLS connections automatically.
The only change required to the app itself is to let it connect to `127.0.0.2:4001` to reach the backend service.
You can find more detailed documentation in the [Service Mesh chapter](components/service-mesh.md).

</TabItem>

<TabItem value="go" label="Go integration">

The mesh certificate contained in `certChain.pem` authenticates this workload, while the mesh CA certificate `mesh-ca.pem` authenticates its peers.
Your app should turn on client authentication to ensure peers are running as confidential containers, too.
See the [Certificate Authority](architecture/certificates.md) section for detailed information about these certificates.

The following example shows how to configure a Golang app, with error handling omitted for clarity.

<Tabs groupId="golang-tls-setup">
<TabItem value="client" label="Client">

```go
caCerts := x509.NewCertPool()
caCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")
caCerts.AppendCertsFromPEM(caCert)
cert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")
cfg := &tls.Config{
  Certificates: []tls.Certificate{cert},
  RootCAs: caCerts,
}
```

</TabItem>
<TabItem value="server" label="Server">

```go
caCerts := x509.NewCertPool()
caCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")
caCerts.AppendCertsFromPEM(caCert)
cert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")
cfg := &tls.Config{
  Certificates: []tls.Certificate{cert},
  ClientAuth: tls.RequireAndVerifyClientCert,
  ClientCAs: caCerts,
}
```

</TabItem>
</Tabs>

</TabItem>
</Tabs>

## Generate policy annotations and manifest

Run the `generate` command to add the necessary components to your deployment files.
This will add the Contrast Initializer to every workload with the specified `contrast-cc` runtime class
and the Contrast Service Mesh to all workloads that have a specified configuration.
After that, it will generate the execution policies and add them as annotations to your deployment files.
A `manifest.json` with the reference values of your deployment will be created.

```sh
contrast generate --reference-values aks-clh-snp resources/
```

:::warning
Please be aware that runtime policies currently have some blind spots. For example, they can't guarantee the starting order of containers. See the [current limitations](features-limitations.md#runtime-policies) for more details.
:::

If you don't want the Contrast Initializer to automatically be added to your
workloads, there are two ways you can skip the Initializer injection step,
depending on how you want to customize your deployment.

<Tabs groupId="injection">
<TabItem value="flag" label="Command-line flag">

You can disable the Initializer injection completely by specifying the
`--skip-initializer` flag in the `generate` command.

```sh
contrast generate --reference-values aks-clh-snp --skip-initializer resources/
```

</TabItem>

<TabItem value="annotation" label="Per-workload annotation">

If you want to disable the Initializer injection for a specific workload with
the `contrast-cc` runtime class, you can do so by adding an annotation to the workload.

```yaml
metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...
  annotations:
    contrast.edgeless.systems/skip-initializer: "true"
```

</TabItem>
</Tabs>

When disabling the automatic Initializer injection, you can manually add the
Initializer as a sidecar container to your workload before generating the
policies. Configure the workload to use the certificates written to the
`contrast-tls-certs` `volumeMount`.

```yaml
# v1.PodSpec
spec:
  initContainers:
    - env:
        - name: COORDINATOR_HOST
          value: coordinator
      image: "ghcr.io/edgelesssys/contrast/initializer:v0.8.0@sha256:9492bcb3534de2ffc9f1127db4d5afaa8bba0db466c483a8628d04463c90f568"
      name: contrast-initializer
      volumeMounts:
        - mountPath: /tls-config
          name: contrast-tls-certs
  volumes:
    - emptyDir: {}
      name: contrast-tls-certs
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
If you can't use a public load balancer, you can deploy a [port-forwarder](https://github.com/edgelesssys/contrast/blob/ddc371b/deployments/emojivoto/portforwarder.yml).
The port-forwarder relays traffic from a CoCo pod and can be accessed via `kubectl port-forward`.

<!-- TODO(burgerdev): inline port-forwarder definition, it has been removed from main. -->

Upstream tracking issue: https://github.com/kata-containers/kata-containers/issues/1693.

:::

## Set the manifest

Attest the Coordinator and set the manifest:

```sh
contrast set -c "${coordinator}:1313" resources/
```

This will use the reference values from the manifest file to attest the Coordinator.
After this step, the Coordinator will start issuing TLS certificates to the workloads. The init container
will fetch a certificate for the workload and the workload is started.

## Verify the Coordinator

An end user (data owner) can verify the Contrast deployment using the `verify` command.

```sh
contrast verify -c "${coordinator}:1313"
```

The CLI will attest the Coordinator using the reference values from the given manifest file. It will then write the
service mesh root certificate and the history of manifests into the `verify/` directory. In addition, the policies
referenced in the active manifest are also written to the directory. The verification will fail if the active
manifest at the Coordinator doesn't match the manifest passed to the CLI.

## Communicate with workloads

You can securely connect to the workloads using the Coordinator's `mesh-ca.pem` as a trusted CA certificate.
First, expose the service on a public IP address via a LoadBalancer service:

```sh
kubectl patch svc ${MY_SERVICE} -p '{"spec": {"type": "LoadBalancer"}}'
kubectl wait --timeout=30s --for=jsonpath='{.status.loadBalancer.ingress}' service/${MY_SERVICE}
lbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo $lbip
```

:::info[Subject alternative names and LoadBalancer IP]

By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed
via load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.
Validation fails since the certificate contains no IP entries as a subject alternative name (SAN).
For example, a connection attempt using the curl and the mesh CA certificate with throw the following error:

```sh
$ curl --cacert ./verify/mesh-ca.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

:::

Using `openssl`, the certificate of the service can be validated with the `mesh-ca.pem`:

```sh
openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```

## Recover the Coordinator

If the Contrast Coordinator restarts, it enters recovery mode and waits for an operator to provide key material.
For demonstration purposes, you can simulate this scenario by deleting the Coordinator pod.

```sh
kubectl delete pod -l app.kubernetes.io/name=coordinator
```

Kubernetes schedules a new pod, but that pod doesn't have access to the key material the previous pod held in memory and can't issue certificates for workloads yet.
You can confirm this by running `verify` again, or you can restart a workload pod, which should stay in the initialization phase.
However, the secret seed in your working directory is sufficient to recover the coordinator.

```sh
contrast recover -c "${coordinator}:1313"
```

Now that the Coordinator is recovered, all workloads should pass initialization and enter the running state.
You can now verify the Coordinator again, which should return the same manifest you set before.

:::warning

The recovery process invalidates the mesh CA certificate:
existing workloads won't be able to communicate with workloads newly spawned.
All workloads should be restarted after the recovery succeeded.

:::
