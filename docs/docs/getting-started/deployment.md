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

Contrast comes with its own PKI infrastructure rooted in attestation.
The Contrast coordinator, an additional service deployed to your cluster, acts as the central attestation service and certificate authority.
It issues certificates to pods that are successfully verified through remote attestation. It can be configured to automatically span a service mesh for pod communication and application authenticity.
The configuration is done via adding special annotations per pod in the deployment files.

In our example we have the following communiction between services:

1. `web` to `emoji` and `voting`: grpc calls should be tunneled via mutual TLS (mTLS) based on mesh certificates.
2. Requests from the outside to `web`: clients use HTTPS to send requests to the frontend but are not required to present a service-mesh certificate themselves. The client will receive the service-mesh certificate of `web` for verification.
3. The `vote-bot` acts as a client simulator and handles HTTPS connections on application level.

For achieving 1., we add a `contrast.edgeless.systems/servicemesh-egress` annotation of the format `<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>` to the pod definition of `web`, and separate multiple entries with ##. Contrast will configure iptables to automatically route any traffic with destination `<original-hostname-or-ip>:<original-port>` through an Envoy proxy running on locally on `<chosen IP>:<chosen port>`. The proxy will establish an mTLS connection with the destination based on the service-mesh certificates.

In our example we set this annotation to `contrast.edgeless.systems/servicemesh-egress: emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080`

For achieving 2., we add `contrast.edgeless.systems/servicemesh-ingress`as an annoation of format `<name>#<port>#false`. This disables requiring client certificates for the service of pod `name` runnnig on `port`. The service-mesh certificate of the `pod` name is still sent to the client as a TLS server certificate for verification.

In our example, we set this to `web#8080#false` to allow HTTPS connections to the frontend.

We further set `contrast.edgeless.systems/expose-service: "true"` to tell Contrast this service is exposed to outside the cluster.

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

## 3. Add Contrast coordinator to deployment

## 4. Generate policy annotations and manifest

## 5. Deploy application

## 6. Verify deployment

## 7. Connect securely to the frontend
