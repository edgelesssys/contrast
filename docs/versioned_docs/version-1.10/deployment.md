# Workload deployment

The following instructions will guide you through the process of making an existing Kubernetes deployment
confidential and deploying it together with Contrast.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/cluster-setup.md) on how to set up a cluster on AKS.
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/bare-metal.md) on how to set up a bare-metal cluster.
</TabItem>
<TabItem value="k3s-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/bare-metal.md) on how to set up a bare-metal cluster.
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
A running CoCo-enabled cluster is required for these steps, see the [setup guide](./getting-started/bare-metal.md) on how to set up a bare-metal cluster.
</TabItem>
</Tabs>

## Deploy the Contrast runtime

Contrast depends on a [custom Kubernetes `RuntimeClass` (`contrast-cc`)](./components/runtime.md),
which needs to be installed in the cluster prior to the Coordinator or any confidential workloads.
This consists of a `RuntimeClass` resource and a `DaemonSet` that performs installation on worker nodes.
This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.10.0/runtime-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.10.0/runtime-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.10.0/runtime-k3s-qemu-snp-gpu.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.10.0/runtime-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>

### Download the Contrast Coordinator resource

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a
LoadBalancer service. Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/download/v1.10.0/coordinator.yml --output-dir deployment
```

## Prepare your Kubernetes resources

Your Kubernetes resources need some modifications to run as Confidential Containers.
This section guides you through the process and outlines the necessary changes.

### Security review

Contrast ensures integrity and confidentiality of the applications, but interactions with untrusted systems require the developers' attention.
Review the [security considerations](architecture/security-considerations.md) and the [certificates](architecture/certificates.md) section for writing secure Contrast application.

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

### Pod resources

Contrast workloads are deployed as one confidential virtual machine (CVM) per pod.
In order to configure the CVM resources correctly, Contrast workloads require a stricter specification of pod resources compared to standard [Kubernetes resource management].

The total memory available to the CVM is calculated from the sum of the individual containers' memory limits and a static `RuntimeClass` overhead that accounts for services running inside the CVM.
Consider the following abbreviated example resource definitions:

```yaml
kind: RuntimeClass
handler: contrast-cc
overhead:
  podFixed:
    memory: 256Mi
---
spec: # v1.PodSpec
  containers:
  - name: my-container
    image: "my-image@sha256:..."
    resources:
      limits:
        memory: 128Mi
  - name: my-sidecar
    image: "my-other-image@sha256:..."
    resources:
      limits:
        memory: 64Mi
```

Contrast launches this pod as a VM with 448MiB of memory: 192MiB for the containers and 256MiB for the Linux kernel, the Kata agent and other base processes.

When calculating the VM resource requirements, init containers aren't taken into account.
If you have an init container that requires large amounts of memory, you need to adjust the memory limit of one of the main containers in the pod.
Since memory can't be shared dynamically with the host, each container should have a memory limit that covers its worst-case requirements.

Kubernetes packs a node until the sum of pod _requests_ reaches the node's total memory.
Since a Contrast pod is always going to consume node memory according to the _limits_, the accounting is only correct if the request is equal to the limit.
Thus, once you determined the memory requirements of your application, you should add a resource section to the pod specification with request and limit:

```yaml
spec: # v1.PodSpec
  containers:
  - name: my-container
    image: "my-image@sha256:..."
    resources:
      requests:
        memory: 50Mi
      limits:
        memory: 50Mi
```

:::note

On bare metal platforms, container images are pulled from within the guest CVM and stored in encrypted memory.
The CVM mounts a `tmpfs` for the image layers that's capped at 50% of the total VM memory.
This tmpfs holds the extracted image layers, so the uncompressed image size needs to be taken into account when setting the container limits.
Registry interfaces often show the compressed size of an image, the decompressed image is usually a factor of 2-4x larger if the content is mostly binary.
For example, the `nginx:stable` image reports a compressed image size of 67MiB, but storing the uncompressed layers needs about 184MiB of memory.
Although only the extracted layers are stored, and those layers are reused across containers within the same pod, the memory limit should account for both the compressed and the decompressed layer simultaneously.
Altogether, setting the limit to 10x the compressed image size should be sufficient for small to medium images.

:::

[Kubernetes resource management]: <https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/>

### Handling TLS

In the initialization process, the `contrast-secrets` shared volume is populated with X.509 certificates for your workload.
These certificates are used by the [Contrast Service Mesh](components/service-mesh.md), but can also be used by your application directly.
The following tab group explains the setup for both scenarios.

<Tabs groupId="tls">
<TabItem value="mesh" label="Drop-in service mesh">

Contrast can be configured to handle TLS in a sidecar container.
This is useful for workloads that are hard to configure with custom certificates, like Java applications.

Configuration of the sidecar depends heavily on the application.
The following example is for an application with these properties:

- The container has a main application at TCP port 8001, which should be TLS-wrapped and doesn't require client authentication.
- The container has a metrics endpoint at TCP port 8080, which should be accessible in plain text.
- All other endpoints require client authentication.
- The app connects to a Kubernetes service `backend.default:4001`, which requires client authentication.

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
caCert, _ := os.ReadFile("/contrast/tls-config/mesh-ca.pem")
caCerts.AppendCertsFromPEM(caCert)
cert, _ := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")
cfg := &tls.Config{
  Certificates: []tls.Certificate{cert},
  RootCAs: caCerts,
}
```

</TabItem>
<TabItem value="server" label="Server">

```go
caCerts := x509.NewCertPool()
caCert, _ := os.ReadFile("/contrast/tls-config/mesh-ca.pem")
caCerts.AppendCertsFromPEM(caCert)
cert, _ := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")
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

### Using GPUs

If the cluster is [configured for GPU usage](./getting-started/bare-metal.md#preparing-a-cluster-for-gpu-usage), Pods can use GPU devices if needed.

To do so, a CDI annotation needs to be added, specifying to use the `pgpu` (passthrough GPU) mode. The `0` corresponds to the PCI device index.
* For nodes with a single GPU, this value is always `0`.
* For nodes with multiple GPUs, the value needs to correspond to the device's order as enumerated on the PCI bus. You can identify this order by inspecting the `/var/run/cdi/nvidia.com-pgpu.yaml` file on the specific node.

This process ensures the correct GPU is allocated to the workload.

As the footprint of a GPU-enabled pod-VM is larger than one of a non-GPU one, the memory of the pod-VM can be adjusted by using the `io.katacontainers.config.hypervisor.default_memory` annotation, which receives the memory the
VM should receive in MiB. The example below sets it to 16 GB. A reasonable minimum for a GPU pod with a light workload is 8 GB.

```yaml
metadata:
  # ...
  annotations:
    # ...
    cdi.k8s.io/gpu: "nvidia.com/pgpu=0"
    io.katacontainers.config.hypervisor.default_memory: "16384"
```

In addition, the container within the pod that requires GPU access must include a device request.
This request specifies the number of GPUs the container should use.
The identifiers for the GPUs, obtained during the [deployment of the NVIDIA GPU Operator](./getting-started/bare-metal.md#preparing-a-cluster-for-gpu-usage), must be included in the request.
In the provided example, the container is allocated a single NVIDIA H100 GPU.

Finally, the environment variable `NVIDIA_VISIBLE_DEVICES` must be set to `all` to grant the container access to GPU utilities provided by the pod-VM. This includes essential tools like CUDA libraries, which are required for running GPU workloads.

```yaml
spec:
  # ...
  containers:
  - # ...
    resources:
      limits:
        "nvidia.com/GH100_H100_PCIE": 1
    env:
    # ...
    - name: NVIDIA_VISIBLE_DEVICES
      value: all
```

:::note
A pod configured to use GPU support may take a few minutes to come up, as the VM creation and boot procedure needs to do more work compared to a non-GPU pod.
:::

## Generate policy annotations and manifest

Run the `generate` command to add the necessary components to your deployment files.
This will add the Contrast Initializer to every workload with the specified `contrast-cc` runtime class
and the Contrast Service Mesh to all workloads that have a specified configuration.
After that, it will generate the [execution policies](components/policies.md) and add them as annotations to your deployment files.
A `manifest.json` with the reference values of your deployment will be created.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
contrast generate --reference-values aks-clh-snp resources/
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
contrast generate --reference-values k3s-qemu-snp resources/
```
:::note[Missing TCB values]
On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models.
:::
</TabItem>
<TabItem value="k3s-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
```sh
contrast generate --reference-values k3s-qemu-snp-gpu resources/
```
:::note[Missing TCB values]
On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models.
:::
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
contrast generate --reference-values k3s-qemu-tdx resources/
```
:::note[Missing TCB values]
On bare-metal TDX, `contrast generate` is unable to fill in the `MinimumTeeTcbSvn` and `MrSeam` TCB values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `ffffffffffffffffffffffffffffffff` and `000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment.
:::
</TabItem>
</Tabs>

The `generate` command needs to pull the container images to derive policies.
Running `generate` for the first time can take a while, especially if the images are large.
If your container registry requires authentication, you can create the necessary credentials with `docker login` or `podman login`.
Be aware of the [registry authentication limitation](features-limitations.md#kubernetes-features) on bare metal.

:::warning
Please be aware that runtime policies currently have some blind spots. For example, they can't guarantee the starting order of containers. See the [current limitations](features-limitations.md#runtime-policies) for more details.
:::

Running `contrast generate` for the first time creates some additional files in the working directory:

* `seedshare-owner.pem` is required for handling the secret seed and recovering the Coordinator (see [Secrets & recovery](architecture/secrets.md)).
* `workload-owner.pem` is required for manifest updates after the initial `contrast set`.
* `rules.rego` and `settings.json` are the basis for [runtime policies](components/policies.md).
* `layers-cache.json` caches container image layer information for your deployments to speed up subsequent runs of `contrast generate`.

If you don't want the Contrast Initializer to automatically be added to your
workloads, there are two ways you can skip the Initializer injection step,
depending on how you want to customize your deployment.

<Tabs groupId="injection">
<TabItem value="flag" label="Command-line flag">

You can disable the Initializer injection completely by specifying the
`--skip-initializer` flag in the `generate` command.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
contrast generate --reference-values aks-clh-snp --skip-initializer resources/
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
contrast generate --reference-values k3s-qemu-snp --skip-initializer resources/
```
</TabItem>
<TabItem value="k3s-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">
```sh
contrast generate --reference-values k3s-qemu-snp-gpu --skip-initializer resources/
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
contrast generate --reference-values k3s-qemu-tdx --skip-initializer resources/
```
</TabItem>
</Tabs>

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
`contrast-secrets` `volumeMount`.

```yaml
# v1.PodSpec
spec:
  initContainers:
    - env:
        - name: COORDINATOR_HOST
          value: coordinator-ready
      image: "ghcr.io/edgelesssys/contrast/initializer:v1.10.0@sha256:c3ef69a9552c8bd1f5ce509742136d7fd8016e3046d676c8f94cc937d0bf1f35"
      name: contrast-initializer
      volumeMounts:
        - mountPath: /contrast
          name: contrast-secrets
  volumes:
    - emptyDir: {}
      name: contrast-secrets
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
For example, attempting to connect with curl and the mesh CA certificate will throw the following error:

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
However, you can recover the Coordinator using the secret seed and the seed share owner key in your working directory.

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
