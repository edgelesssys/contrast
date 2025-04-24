# Workload deployment

## 1. Adjust deployment files

### Download demo deployment configuration

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/emojivoto-demo.yml --create-dirs --output-dir deployment
```

### Adjust RuntimeClass

For each pod configuration adjust the `RuntimeClassName` in `spec` to `contrast-cc`:

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

### Adjust pod resources

Contrast workloads are deployed as one CVM per pod. Contrast workloads require stricter specification of pod resources compared to standard Kubernetes resource management.

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

### Add service-mesh annotations

Contrast comes with its own PKI infrastructure, rooted in attestation.

The **Contrast Coordinator**, an additional service deployed to your cluster, acts as both the **central attestation service** and a **certificate authority**. It issues certificates only to pods that have been successfully verified through remote attestation. It can also be configured to automatically establish a service mesh that ensures authenticated and encrypted pod-to-pod communication.

This configuration is done by adding specific annotations to each pod in the deployment files.

In our setup, the communication between services works as follows:

1. **`web` to `emoji` and `voting`**:
   gRPC calls are tunneled via mutual TLS (mTLS), using service mesh certificates.

2. **External requests to `web`**:
   Clients use HTTPS to send requests to the frontend. They are not required to present a service mesh certificate. During the TLS handshake, the client receives the service mesh certificate of `web` for verification.

3. **`vote-bot` as a client simulator**:
   This component acts as a simulated client and initiates HTTPS connections at the application level.

#### Enabling mTLS Egress for `web`

To enable secure, authenticated egress from `web` to `emoji` and `voting`, we add the following annotation:

```yaml
contrast.edgeless.systems/servicemesh-egress: emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
```

The format is:

```
<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>
```

- `<name>` is an internal label for the target service.
- `<chosen IP>:<chosen port>` is a local loopback address the application will use.
- `<original-hostname-or-ip>:<original-port>` is the real destination that traffic should reach.

Multiple entries are separated by `##`.

Contrast configures `iptables` rules so that any traffic targeting `<original-hostname-or-ip>:<original-port>` is transparently redirected to the Envoy proxy running on `<chosen IP>:<chosen port>` inside the same pod. The proxy then establishes a mutual TLS connection with the actual destination, using the pod's service mesh certificate for authentication.

#### Configuring Ingress for `web`

To allow external HTTPS connections without requiring client certificates, we add the following annotation:

```yaml
contrast.edgeless.systems/servicemesh-ingress: web#8080#false
```

This configures the `web` pod to:

- Accept HTTPS traffic on port `8080`,
- Not require clients to present a certificate (`false`),
- Still present its own service mesh certificate as the server certificate, allowing clients to verify the pod's identity.

#### Exposing the Service to the Outside

Finally, we add the annotation:

```yaml
contrast.edgeless.systems/expose-service: "true"
```

This tells Contrast that the service is exposed externally (e.g., via a `LoadBalancer`) and enables Contrast to handle TLS termination for inbound connections.

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
@@ -217,6 +224,8 @@
 apiVersion: v1
 kind: Service
 metadata:
+  annotations:
+    contrast.edgeless.systems/expose-service: "true"
   name: web-svc
 spec:
   ports:
```

## 2. Setup Contrast runtime

After adjusting the deployment files, we add the Contrast runtime to the deployment. The runtime takes care of setting up CVMs on nodes.

This step is only required once for each version of the runtime.
It can be shared between Contrast deployments.

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

## 3. Add Contrast coordinator to deployment

Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a
LoadBalancer service. Put it next to your resources:

```sh
curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir deployment
```

## 4. Generate policy annotations and manifest

Run the `generate` command to generate the execution policies and add them as
annotations to your deployment files. A `manifest.json` file with the reference values
of your deployment will be created:

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
On bare-metal SEV-SNP, `contrast generate` is unable to fill in the `MinimumTCB` values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}` and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models.
:::
</TabItem>
<TabItem value="k3s-qemu-tdx" label="Bare metal (TDX)">
```sh
contrast generate --reference-values k3s-qemu-tdx deployment/
```
:::note[Missing TCB values]
On bare-metal TDX, `contrast generate` is unable to fill in the `MinimumTeeTcbSvn` and `MrSeam` TCB values as they can vary between platforms.
They will have to be filled in manually.
If you don't know the correct values use `ffffffffffffffffffffffffffffffff` and `000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment.
:::
</TabItem>
</Tabs>

## 5. Deploy application

```sh
kubectl apply -f deployment/
```

## 6. Connect to Coordinator and set manifest

Configure the Coordinator with a manifest. It might take up to a few minutes
for the load balancer to be created and the Coordinator being available.

```sh
coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "The user API of your Contrast Coordinator is available at $coordinator:1313"
contrast set -c "${coordinator}:1313" deployment/
```

The CLI will use the reference values from the manifest to attest the Coordinator deployment
during the TLS handshake. If the connection succeeds, it's ensured that the Coordinator
deployment hasn't been tampered with.

## 6. Verify deployment

n different scenarios, users of an app may want to verify its security and identity before sharing data, for example, before casting a vote.
With Contrast, a user only needs a single remote-attestation step to verify the deployment - regardless of the size or scale of the deployment.
Contrast is designed such that, by verifying the Coordinator, the user transitively verifies those systems the Coordinator has already verified or will verify in the future.
Successful verification of the Coordinator means that the user can be sure that the given manifest will be enforced.

### Verifying the Coordinator

A user can verify the Contrast deployment using the verify
command:

```sh
contrast verify -c "${coordinator}:1313" -m manifest.json
```

The CLI will verify the Coordinator via remote attestation using the reference values from a given manifest. This manifest needs
to be communicated out of band to everyone wanting to verify the deployment, as the `verify` command checks
if the currently active manifest at the Coordinator matches the manifest given to the CLI. If the command succeeds,
the Coordinator deployment was successfully verified to be running in the expected Confidential
Computing environment with the expected code version. The Coordinator will then return its
configuration over the established TLS channel. The CLI will store this information, namely the root
certificate of the mesh (`mesh-ca.pem`) and the history of manifests, into the `verify/` directory.
In addition, the policies referenced in the manifest history are also written into the same directory.

### Auditing the manifest history and artifacts

In the next step, the Coordinator configuration that was written by the `verify` command needs to be audited.
A potential voter should inspect the manifest and the referenced policies. They could delegate
this task to an entity they trust.

## 7. Connect securely to the frontend

After ensuring the configuration of the Coordinator fits the expectation, the user can securely connect
to the application using the Coordinator's `mesh-ca.pem` as a trusted CA certificate.

To access the web frontend, expose the service on a public IP address via a LoadBalancer service:

```sh
frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Frontend is available at  https://$frontendIP, you can visit it in your browser."
```

Using `openssl`, the certificate of the service can be validated with the `mesh-ca.pem`:

```sh
openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```
