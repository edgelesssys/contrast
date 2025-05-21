# Workload deployment

Follow these steps to make your deployment confidential.

## 1. Adjust deployment files

### Download the demo deployment configuration

Start by downloading the initial, non-confidential deployment file for the emojivoto demo:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/emojivoto-demo.yml --create-dirs --output-dir deployment
```

<!-- TODO: Ajust YAML to vanilla  -->

### Set the RuntimeClass

In each pod configuration, set the `runtimeClassName` field under `spec` to `contrast-cc`.
This tells Kubernetes to run the pod inside a Confidential Virtual Machine (CVM) using the Contrast runtime.

```diff title="deployment/emojivoto-demo.yaml"
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

### Define pod resources

Each Contrast workload runs inside its own Confidential Virtual Machine (CVM).
To ensure accurate memory allocation, Contrast requires **strict resource definitions**:

* Always specify both memory `requests` and `limits`.
* The values for `requests` and `limits` **must be identical**.
* Only set limits on **main containers**. Limits for `initContainers` and sidecars are **ignored**.

What matters is that the **sum of memory limits across all main containers** in a pod is as least as large as `sum_of_limits`. With `sum_of_limits` computed as follows:

<Tabs queryString="platform">

<TabItem value="aks-clh-snp" label="AKS" default>

```
sum_of_limits = worst-case memory requirement of all simultaneous container processes
```

For example, if an `initContainer` requires more memory than the combined usage of all main containers and sidecars, ensure that the **sum of limits for all main containers** is **at least as large** as the `initContainer`'s memory requirement.

</TabItem>

<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">

```
sum_of_limits = worst-case memory requirement of all simultaneous container processes
```

For example, if an `initContainer` requires more memory than the combined usage of all main containers and sidecars, ensure that the **sum of limits for all main containers** is **at least as large** as the `initContainer`'s memory requirement.

In addition, account for the total unpacked size of all container images:

```
sum_of_limits += unpacked image sizes of all main containers, initContainers, and sidecars
```

</TabItem>

<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">

```
sum_of_limits = worst-case memory requirement of all simultaneous container processes
```

For example, if an `initContainer` requires more memory than the combined usage of all main containers and sidecars, ensure that the **sum of limits for all main containers** is **at least as large** as the `initContainer`'s memory requirement.

In addition, account for the total unpacked size of all container images:

```
sum_of_limits += unpacked image sizes of all main containers, initContainers, and sidecars
```

</TabItem>

</Tabs>

For details, see our [pod resources how-to](../howto/workload-deployment/deployment-file-preparation.md#pod-resources).

If you define a lower memory `request` than `limit`, Kubernetes may overcommit memory, potentially leading to pod failures.
To avoid this, set both values equally. Set both to 700Mi for each pod in this example:

```diff title="deployment/emojivoto-demo.yaml"
@@ -36,6 +36,11 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+          resources:
+            limits:
+              memory: 700Mi
+            requests:
+              memory: 700Mi
       runtimeClassName: contrast-cc
       serviceAccountName: emoji
 ---
@@ -87,6 +92,11 @@
               value: web-svc:443
           image: ghcr.io/edgelesssys/contrast/emojivoto-web:coco-1@sha256:0fd9bf6f7dcb99bdb076144546b663ba6c3eb457cbb48c1d3fceb591d207289c
           name: vote-bot
+          resources:
+            limits:
+              memory: 700Mi
+            requests:
+              memory: 700Mi
       runtimeClassName: contrast-cc
 ---
 apiVersion: apps/v1
@@ -127,6 +137,11 @@
             periodSeconds: 5
             tcpSocket:
               port: 8080
+          resources:
+            limits:
+              memory: 700Mi
+            requests:
+              memory: 700Mi
       runtimeClassName: contrast-cc
       serviceAccountName: voting
 ---
@@ -191,6 +206,11 @@
               scheme: HTTPS
             initialDelaySeconds: 1
             periodSeconds: 5
+          resources:
+            limits:
+              memory: 700Mi
+            requests:
+              memory: 700Mi
       runtimeClassName: contrast-cc
       serviceAccountName: web
 ---
```

### Add service mesh annotations

Contrast provides its own PKI based on attestation.
The **Contrast Coordinator**, a service deployed alongside your workloads, acts as both:

- The central remote attestation service.
- A certificate authority (CA) that issues certificates to verified pods.

By adding specific annotations to your pod definitions, you enable an automatic service mesh for encrypted and authenticated pod-to-pod communication.

#### Traffic flow in this setup:

1. **From `web` to `emoji` and `voting`:**
   gRPC calls are tunneled via mTLS using mesh certificates.

2. **From external clients to `web`:**
   HTTPS is used. Clients don't need to present a mesh certificate but can verify `web`’s certificate.

3. **`vote-bot` as a simulated client:**
   Simulates external clients with HTTPS requests.

#### Enable egress mTLS from `web` to `emoji` and `voting`

Add the following annotation to the `web` pod to define egress rules:

```yaml
contrast.edgeless.systems/servicemesh-egress: emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
```

The format of this annotation is:

```
<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>
```

Where:

- `<name>` is an internal label for the target service.
- `<chosen IP>:<chosen port>` is a local address that your application uses.
- `<original-hostname-or-ip>:<original-port>` is the actual destination the traffic should reach.

Multiple entries are separated by `##`.

Contrast’s service mesh deploys a local Envoy proxy at `<chosen IP>:<chosen port>` within each pod. To direct outbound traffic through the mesh, you must configure your application to connect to that proxy address instead of `<original-hostname-or-ip>:<original-port>`.


#### Enable ingress mTLS at `emoji` and `voting`

Add the following annotation to both pods:

```yaml
contrast.edgeless.systems/servicemesh-ingress: ""
```

Setting this annotation to an empty string enables automatic verification of incoming mTLS connections.
If the annotation is omitted entirely, no ingress verification will take place.

#### Enable ingress for external HTTPS traffic

Add this annotation to allow HTTPS ingress without requiring client certificates:

```yaml
contrast.edgeless.systems/servicemesh-ingress: web#8080#false
```
This configuration exempts port 8080 from mTLS verification: clients can connect to 8080 without presenting a client certificate, while all other ports still require full mTLS


Altogether, we can configure the service mesh by adding the following annotations:

```diff title="deployment/emojivoto-demo.yaml"
@@ -1,6 +1,8 @@
 apiVersion: apps/v1
 kind: Deployment
 metadata:
+  annotations:
+    contrast.edgeless.systems/servicemesh-ingress: ""
   labels:
     app.kubernetes.io/name: emoji
     app.kubernetes.io/part-of: emojivoto
@@ -102,6 +104,8 @@
 apiVersion: apps/v1
 kind: Deployment
 metadata:
+  annotations:
+    contrast.edgeless.systems/servicemesh-ingress: ""
   labels:
     app.kubernetes.io/name: voting
     app.kubernetes.io/part-of: emojivoto
@@ -168,6 +172,9 @@
 apiVersion: apps/v1
 kind: Deployment
 metadata:
+  annotations:
+    contrast.edgeless.systems/servicemesh-egress: emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
+    contrast.edgeless.systems/servicemesh-ingress: web#8080#false
   labels:
     app.kubernetes.io/name: web
     app.kubernetes.io/part-of: emojivoto
```

These are all the changes you need to make to your deployment files.

## 2. Install the Contrast runtime

Next, install the Contrast runtime in your cluster which will be used when setting up CVMs on nodes.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-aks-clh-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp.yml
```
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-tdx.yml
```
</TabItem>
</Tabs>

## 3. Add the Contrast Coordinator to the deployment

The Contrast Coordinator is an additional service that runs alongside your application and ensures the deployment remains in a secure and trusted state by attesting all workload pods.

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a LoadBalancer service.
Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir deployment
```

## 4. Generate policy annotations and manifest

Run the `generate` command to create execution policies that strictly control communication between the host and CVMs on each worker node and define which workloads are allowed to run. These policies are automatically added as annotations to your deployment files.

The command also generates a `manifest.json` file, which contains the trusted reference values of your deployment.

<Tabs queryString="platform">
<TabItem value="aks-clh-snp" label="AKS" default>
```sh
contrast generate --reference-values aks-clh-snp deployment/
```
</TabItem>
<TabItem value="k3s-qemu-snp" label="Bare metal (SEV-SNP)">
```sh
contrast generate --reference-values k3s-qemu-snp deployment/
```
:::note[Missing TCB values]
On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values of the created manifest.json as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models.
:::
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
contrast generate --reference-values k3s-qemu-tdx deployment/
```
:::note[Missing TCB values]
On bare-metal TDX, `contrast generate` is unable to fill in the `MinimumTeeTcbSvn` and `MrSeam` TCB values of the created manifest.json as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `ffffffffffffffffffffffffffffffff` and `000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` respectively and observe the accurate ones in the error messages in the following steps. This should only be done in a secure environment.
:::
</TabItem>
</Tabs>

## 5. Deploy application

Now deploy your application along with the Contrast coordinator:

```sh
kubectl apply -f deployment/
```

## 6. Connect to Coordinator and set manifest

Configure the Coordinator with the created manifest. It might take up to a few minutes
for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" deployment/
```
The CLI will use the reference values from the manifest to attest the Coordinator deployment
during the TLS handshake. If the connection succeeds, it's ensured that the Coordinator
deployment hasn't been tampered with.

## 7. Verify deployment

In many scenarios, users may require assurance of an application's security and integrity before interacting with it—for example, prior to submitting sensitive data or casting a vote.

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
The CLI stores this information—including the mesh root certificate (`mesh-ca.pem`), the manifest history, and the associated policies—in the `verify/` directory.

### Auditing the manifest and policies

Next, the stored Coordinator configuration should be audited.
A user—or a trusted third party—can review the manifest and the referenced policies to ensure they meet expectations.

## 8. Connect securely to the frontend

Once the Coordinator’s configuration has been verified, users can securely connect to the application via HTTPS. The application uses the `mesh-ca.pem` certificate as the root of trust. We can use this certificate to validate the frontend's certificate.

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
Tools like `curl` validate certificates against SANs and will fail if the certificate doesn’t include the IP address.

For example, running `curl` with the mesh CA certificate results in:

```sh
$ curl --cacert ./verify/mesh-ca.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

### Adding an IP SAN to the manifest

To enable IP-based certificate verification, update the relevant policy in the manifest.
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
       "WorkloadSecretID": "web"
     },
```

### Updating the manifest on the coordinator

Apply the updated manifest with:

```sh
contrast set -c "${coordinator}:1313" deployment/
```

This triggers a rotation of the mesh CA certificate.
New certificates will be issued by the updated CA.
Previously issued certificates will no longer be trusted.
This ensures that updated workloads don’t trust older, potentially vulnerable versions.

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
