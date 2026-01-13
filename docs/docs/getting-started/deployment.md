# Workload deployment

Follow these steps to make your deployment confidential.

## 1. Adjust deployment files

### Download the demo deployment configuration

Download the Kubernetes resources for the emojivoto deployment with the following command:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/emojivoto-demo.yml --create-dirs --output-dir resources
```

This deployment has already been prepared for Contrast.

<details>
<summary>
What was modified compared to the original deployment file?
</summary>

In the following sub-sections, we will explain the changes made to the original deployment file.

### Set the RuntimeClass

In each pod configuration, set the `runtimeClassName` field under `spec` to `contrast-cc`.
This tells Kubernetes to run the pod inside a Confidential Virtual Machine (CVM) using the Contrast runtime.

```diff title="resources/emojivoto-demo.yaml"
@@ -36,6 +36,7 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+      runtimeClassName: contrast-cc
       serviceAccountName: emoji
 ---
 apiVersion: v1
@@ -86,6 +87,7 @@
               value: web-svc:443
           image: ghcr.io/edgelesssys/contrast/emojivoto-web:coco-1@sha256:0fd9bf6f7dcb99bdb076144546b663ba6c3eb457cbb48c1d3fceb591d207289c
           name: vote-bot
+      runtimeClassName: contrast-cc
 ---
 apiVersion: apps/v1
 kind: Deployment
@@ -125,6 +127,7 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+      runtimeClassName: contrast-cc
       serviceAccountName: voting
 ---
 apiVersion: v1
@@ -188,6 +191,7 @@
               scheme: HTTPS
             initialDelaySeconds: 1
             periodSeconds: 5
+      runtimeClassName: contrast-cc
       serviceAccountName: web
 ---
 apiVersion: v1
```

For more details, see [RuntimeClass Section in the "Prepare deployment files" how-to](../howto/workload-deployment/deployment-file-preparation.md#runtimeclass).

### Define pod resources

Each Contrast workload runs inside its own CVM.
To ensure accurate memory allocation, Contrast uses the memory `limits` defined by a pod.
You should set appropriate memory `limits` for each container to get a VM of the right size.

On bare-metal platforms, the container images are pulled from within each pod-VM.
By default, images are stored in encrypted memory, but can optionally also be stored on an encrypted ephemeral disk through the [Contrast secure image store](../howto/secure-image-store.md) feature.
In the latter case, the image sizes don't need to be taken into account when calculating the memory limits of containers.

Kubernetes schedules pods on nodes based on the memory `requests`.
To prevent Kubernetes from over-commiting nodes, set the memory `request` to the same value as the `limit`.

<!-- TODO(katexochen): Show the full calculation for the emojivoto example, show how we arrive at 700Mi -->

Set both to 400Mi for each pod in this example:

```diff title="resources/emojivoto-demo.yaml"
@@ -36,6 +36,11 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+          resources:
+            limits:
+              memory: 400Mi
+            requests:
+              memory: 400Mi
       runtimeClassName: contrast-cc
       serviceAccountName: emoji
 ---
@@ -87,6 +92,11 @@
               value: web-svc:443
           image: ghcr.io/edgelesssys/contrast/emojivoto-web:coco-1@sha256:0fd9bf6f7dcb99bdb076144546b663ba6c3eb457cbb48c1d3fceb591d207289c
           name: vote-bot
+          resources:
+            limits:
+              memory: 400Mi
+            requests:
+              memory: 400Mi
       runtimeClassName: contrast-cc
 ---
 apiVersion: apps/v1
@@ -127,6 +137,11 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+          resources:
+            limits:
+              memory: 400Mi
+            requests:
+              memory: 400Mi
       runtimeClassName: contrast-cc
       serviceAccountName: voting
 ---
@@ -191,6 +206,11 @@
               scheme: HTTPS
             initialDelaySeconds: 1
             periodSeconds: 5
+          resources:
+            limits:
+              memory: 400Mi
+            requests:
+              memory: 400Mi
       runtimeClassName: contrast-cc
       serviceAccountName: web
 ---
```

For more details, see the [Pod Resources Section in the "Prepare deployment files" how-to](../howto/workload-deployment/deployment-file-preparation#pod-resources).

### Add service mesh annotations

Contrast provides its own PKI based on attestation.
The **Contrast Coordinator**, a service deployed alongside your workloads, acts as both:

- The central remote attestation service.
- A certificate authority (CA) that issues certificates to verified pods.

By adding specific annotations to your pod definitions, you enable an automatic service mesh for encrypted and authenticated pod-to-pod communication.

For more details, see the [Drop-in service mesh Section in the "TLS Configuration" how-to](../howto/workload-deployment/TLS-configuration#drop-in-service-mesh).

#### Traffic flow in this setup:

1. **From `web` to `emoji` and `voting`:**
   gRPC calls are tunneled via mTLS using mesh certificates.

2. **From external clients to `web`:**
   HTTPS is used. Clients don't need to present a mesh certificate but can verify `web`'s certificate.

3. **`vote-bot` as a simulated client:**
   Simulates external clients with HTTPS requests.

#### Enable egress mTLS from `web` to `emoji` and `voting`

Add the following annotation to the `web` pod to define egress rules:

```yaml
contrast.edgeless.systems/servicemesh-egress: emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
```

Contrast's service mesh deploys a local Envoy proxy for each pod.
The above rule will direct outbound traffic through the mesh by configuring the application to direct traffic to that proxy at `127.137.0.1:8081` instead of `emoji-svc:8080` as well as to `127.137.0.2:8081` instead of `voting-svc:8080`.

For more details on the format, see the [Annotation format Section in the "Service Mesh" Chapter](../architecture/components/service-mesh#annotation-format).

#### Enable ingress mTLS at `emoji` and `voting`

Add the following annotation to both pods:

```yaml
contrast.edgeless.systems/servicemesh-ingress: ""
```

Setting this annotation to an empty string enables automatic verification of incoming mTLS connections.
If the annotation is omitted entirely, no ingress verification will take place.


#### Enable ingress for external HTTPS traffic

Add this annotation to the `web` pod to allow HTTPS ingress without requiring client certificates:

```yaml
contrast.edgeless.systems/servicemesh-ingress: web#8080#false
```

This configuration exempts port 8080 from mTLS verification: clients can connect to 8080 without presenting a client certificate, while all other ports still require full mTLS

#### Enable egress mTLS from `vote-bot` to `web`

Add the following annotation to the `vote-bot` pod to define egress rules:

```yaml
contrast.edgeless.systems/servicemesh-egress: web#127.137.0.3:8081#web-svc:443
contrast.edgeless.systems/servicemesh-ingress: DISABLED
```

Presence of either one of the annotations will enable the service mesh for both ingress and egress. The ingress annotation supports the option `DISABLED` to explicitly disable it.


Altogether, we can configure the service mesh by adding the following annotations:

```diff title="resources/emojivoto-demo.yml"
@@ -14,6 +14,8 @@ spec:
       version: v11
   template:
     metadata:
+      annotations:
+        contrast.edgeless.systems/servicemesh-ingress: ""
       labels:
         app.kubernetes.io/name: emoji-svc
         version: v11
@@ -74,6 +76,9 @@ spec:
       version: v11
   template:
     metadata:
+      annotations:
+        contrast.edgeless.systems/servicemesh-egress: "web#127.137.0.3:8081#web-svc:443"
+        contrast.edgeless.systems/servicemesh-ingress: DISABLED
       labels:
         app.kubernetes.io/name: vote-bot
         version: v11
@@ -105,6 +110,8 @@ spec:
       version: v11
   template:
     metadata:
+      annotations:
+        contrast.edgeless.systems/servicemesh-ingress: ""
       labels:
         app.kubernetes.io/name: voting-svc
         version: v11
@@ -165,6 +172,9 @@ spec:
       version: v11
   template:
     metadata:
+      annotations:
+        contrast.edgeless.systems/servicemesh-egress: "emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080"
+        contrast.edgeless.systems/servicemesh-ingress: "web#8080#false"
       labels:
         app.kubernetes.io/name: web-svc
         version: v11
```

These are all the changes you need to make to your deployment files.

</details>

## 2. Install the Contrast runtime

Next, install the Contrast runtime in your cluster which will be used when setting up CVMs on nodes.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-snp.yml
```
</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-metal-qemu-tdx.yml
```
</TabItem>
</Tabs>

The contrast runtime is deployed in the `contrast-system` namespace by default but this can be changed by modifying the YAML file before applying if you wish.

For more details, see the ["Deploy the Contrast runtime" how-to](../howto/workload-deployment/runtime-deployment).

## 3. Add the Contrast Coordinator to the deployment

The Contrast Coordinator is an additional service that runs alongside your application and ensures the deployment remains in a secure and trusted state by attesting all workload pods.

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a LoadBalancer service.
Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir resources
```

## 4. Generate initdata annotations and manifest

Run the `generate` command to create execution policies that strictly control communication between the host and CVMs on each worker node and define which workloads are allowed to run. These policies are wrapped in initdata documents and added as annotations to your deployment files.

The command also generates a `manifest.json` file, which contains the trusted reference values of your deployment.

<Tabs queryString="platform">
<TabItem value="metal-qemu-snp" label="Bare metal (SEV-SNP)">

```sh
contrast generate --reference-values metal-qemu-snp resources/
```

On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms and CPU models.
They will have to be filled in manually.

AMD doesn't provide an accessible way to acquire the latest TCB values for your platform.
Visit the [AMD SEV developer portal](https://www.amd.com/en/developer/sev.html) and download the latest firmware package for your processor family.
Unpack and inspect the contained release notes, which state the SNP firmware SVN (called `SPL` (security patch level) in that document).
Contact your hardware vendor or BIOS firmware provider for information about the other TCB components.

To check the current TCB level of your platform, use the [`snphost`](https://github.com/virtee/snphost):

```sh
snphost show tcb
```
```console
Reported TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
Platform TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
```

The values listed as `Reported TCB` to should be greater or equal to the `MinimumTCB` values in `manifest.json`.
The `Platform TCB` can be higher than the `Reported TCB`, in this case, the platform has provisional firmware enrolled.
Contrast relies on the committed TCB values, as provisional firmware can be rolled back anytime by the platform operator.

:::warning

The TCB values observed on the target platform using `snphost` might not be trustworthy.
Your channel to the system or the system itself might be compromised.
The deployed firmware could be outdated and vulnerable.

:::

</TabItem>
<TabItem value="metal-qemu-snp-gpu" label="Bare metal (SEV-SNP, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-snp-gpu resources/
```

On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms and CPU models.
They will have to be filled in manually.

AMD doesn't provide an accessible way to acquire the latest TCB values for your platform.
Visit the [AMD SEV developer portal](https://www.amd.com/en/developer/sev.html) and download the latest firmware package for your processor family.
Unpack and inspect the contained release notes, which state the SNP firmware SVN (called `SPL` (security patch level) in that document).
Contact your hardware vendor or BIOS firmware provider for information about the other TCB components

To check the current TCB level of your platform, use the [`snphost`](https://github.com/virtee/snphost):

```sh
snphost show tcb
```
```console
Reported TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
Platform TCB: TCB Version:
  Microcode:   72
  SNP:         23
  TEE:         0
  Boot Loader: 9
  FMC:         None
```

The values listed as `Reported TCB` to should be greater or equal to the `MinimumTCB` values in `manifest.json`.
The `Platform TCB` can be higher than the `Reported TCB`, in this case, the platform has provisional firmware enrolled.
Contrast relies on the committed TCB values, as provisional firmware can be rolled back anytime by the platform operator.

:::warning

The TCB values observed on the target platform using `snphost` might not be trustworthy.
Your channel to the system or the system itself might be compromised.
The deployed firmware could be outdated and vulnerable.

:::

</TabItem>
<TabItem value="metal-qemu-tdx" label="Bare metal (TDX)">

```sh
contrast generate --reference-values metal-qemu-tdx resources/
```

On bare-metal TDX, `contrast generate` is unable to fill in the `MrSeam` value as it depends on your platform configuration.
It will have to be filled in manually.

`MrSeam` is the SHA384 hash of the TDX module.
You should retrieve the TDX module via a trustworthy channel from Intel, for example by downloading the TDX module [from Intel's GitHub repository](https://github.com/intel/confidential-computing.tdx.tdx-module/releases) and hashing the module on a trusted machine.
You can also reproduce the release artifact by following the build instructions linked in the release notes.

You can check the hash of the in-use TDX module by executing

```sh
sha384sum /boot/efi/EFI/TDX/TDX-SEAM.so | cut -d' ' -f1
```

:::warning

The TDX module hash (`MrSeam`) observed on the target platform might not be trustworthy.
Your channel to the system or the system itself might be compromised.
Make sure to retrieve or reproduce the value on a trusted machine.

:::

</TabItem>
<TabItem value="metal-qemu-tdx-gpu" label="Bare metal (TDX, with GPU support)">

```sh
contrast generate --reference-values metal-qemu-tdx-gpu resources/
```

On bare-metal TDX, `contrast generate` is unable to fill in the `MrSeam` value as it depends on your platform configuration.
It will have to be filled in manually.

`MrSeam` is the SHA384 hash of the TDX module.
You should retrieve the TDX module via a trustworthy channel from Intel, for example by downloading the TDX module [from Intel's GitHub repository](https://github.com/intel/confidential-computing.tdx.tdx-module/releases) and hashing the module on a trusted machine.
You can also reproduce the release artifact by following the build instructions linked in the release notes.

You can check the hash of the in-use TDX module by executing

```sh
sha384sum /boot/efi/EFI/TDX/TDX-SEAM.so | cut -d' ' -f1
```

:::warning

The TDX module hash (`MrSeam`) observed on the target platform might not be trustworthy.
Your channel to the system or the system itself might be compromised.
Make sure to retrieve or reproduce the value on a trusted machine.

:::

</TabItem>
</Tabs>

## 5. Deploy application

Now deploy your application along with the Contrast coordinator:

```sh
kubectl apply -f resources/
```

The Coordinator should show a status `Running` and will only transition to `Ready` after a manifest has been set.

## 6. Connect to Coordinator and set manifest

Configure the Coordinator with the created manifest. It might take up to a few minutes
for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" resources/
```

The CLI will use the reference values from the manifest to attest the Coordinator deployment
during the TLS handshake. If the connection succeeds, it's ensured that the Coordinator
deployment hasn't been tampered with.

## 7. Verify deployment

In many scenarios, users may require assurance of an application's security and integrity before interacting with it, for example, prior to submitting sensitive data or casting a vote.

Contrast enables this through a single remote attestation step, regardless of the size or complexity of the deployment.
By attesting the Coordinator, the user transitively attests all workloads that the Coordinator has verified or will verify according to the defined manifest.

A successful attestation of the Coordinator provides a strong guarantee that the deployment adheres to the reference values specified in the `manifest.json`.

### Verifying the Coordinator

A user can verify the Contrast deployment using the verify
command:

```sh
contrast verify -c "${coordinator}:1313" -m manifest.json
```

The CLI verifies the Coordinator via remote attestation using reference values from the provided manifest.
This manifest must be distributed out-of-band to all parties performing verification, as the `verify` command checks whether the manifest currently active at the Coordinator matches the one supplied to the CLI.

If verification succeeds, it confirms that the Coordinator is running in the expected Confidential Computing environment with the correct code version.
The Coordinator then returns its configuration over a secure TLS channel.
The CLI stores this information, including the mesh root certificate (`mesh-ca.pem`), the manifest history, and the associated initdata documents, in the `verify/` directory.

### Auditing the manifest

Next, the stored Coordinator configuration should be audited.
A user, or a trusted third party, can review the manifest and the referenced initdata documents to ensure they meet expectations.

## 8. Connect securely to the frontend

Once the Coordinator's configuration has been verified, users can securely connect to the application via HTTPS. The application uses the `mesh-ca.pem` certificate as the root of trust. We can use this certificate to validate the frontend's certificate.

To access the web frontend, expose the service on a public IP address via a LoadBalancer service:

```sh
frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Frontend is available at  https://$frontendIP, you can visit it in your browser."
```

Using `openssl`, the certificate of the service can be validated with the `mesh-ca.pem`:

```sh
openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```

## Optional: Updating the certificate SAN and the manifest

By default, mesh certificates are issued with a wildcard DNS Subject Alternative Name (SAN).
In this demo, the web frontend is accessed via a LoadBalancer IP.
Tools like `curl` validate certificates against SANs and will fail if the certificate doesn't include the IP address.

For example, running `curl` with the mesh CA certificate results in:

```sh
$ curl --cacert ./verify/mesh-ca.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

### Adding an IP SAN to the manifest

To enable IP-based certificate verification, update the relevant entry in the manifest.
Add the `frontendIP` to the list of SANs:

```diff
   "Policies": {
     ...
     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {
       "SANs": [
         "web",
-        "*"
+        "*",
+        "203.0.113.34"
       ],
     },
```

To find out more about the `SANs` field in the manifest, see [SANs section in the manifest reference](../architecture/components/manifest.md#policies-sans).

### Updating the manifest on the coordinator

Apply the updated manifest with:

```sh
contrast set -c "${coordinator}:1313" deployment/
```

This triggers a rotation of the mesh CA certificate.
New certificates will be issued by the updated CA.
Previously issued certificates will no longer be trusted.
This ensures that updated workloads don't trust older, potentially vulnerable versions.

The updated `mesh-ca.pem` will be written to reflect the new CA.

### Restarting workloads

The Contrast Initializer doesn't automatically fetch new certificates after a manifest update.
You must manually restart the deployments to trigger certificate re-issuance:

```sh
kubectl rollout restart deployment/emoji
kubectl rollout restart deployment/vote-bot
kubectl rollout restart deployment/voting
kubectl rollout restart deployment/web
```

### Verifying the connection

After restarting the deployments, connect securely using:

```sh
curl --cacert ./mesh-ca.pem "https://${frontendIP}:443"
```

This should return the HTML content of the web frontend.
