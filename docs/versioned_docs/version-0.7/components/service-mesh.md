# Service Mesh

The Contrast service mesh secures the communication of the workload by automatically
wrapping the network traffic inside mutual TLS (mTLS) connections. The
verification of the endpoints in the connection establishment is based on
certificates that are part of the
[PKI of the Coordinator](../architecture/certificates.md).

The service mesh can be enabled on a per-pod basis by adding the `service-mesh`
container as a [sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/).
The service mesh container first sets up `iptables`
rules based on its configuration and then starts [Envoy](https://www.envoyproxy.io/)
for TLS origination and termination.

## Configuring the Proxy

The service mesh container can be configured using the `EDG_INGRESS_PROXY_CONFIG`
and `EDG_EGRESS_PROXY_CONFIG` environment variables.

### Ingress

All TCP ingress traffic is routed over Envoy by default. Since we use
[TPROXY](https://docs.kernel.org/networking/tproxy.html), the destination address
remains the same throughout the packet handling.

Any incoming connection is required to present a client certificate signed by the
[mesh CA certificate](../architecture/certificates.md#usage-of-the-different-certificates).
Envoy presents a certificate chain of the mesh
certificate of the workload and the intermediate CA certificate as the server certificate.

If the deployment contains workloads which should be reachable from outside the
Service Mesh, while still handing out the certificate chain, disable client
authentication by setting the environment variable `EDG_INGRESS_PROXY_CONFIG` as
`<name>#<port>#false`. Separate multiple entries with `##`. You can choose any
descriptive string identifying the service on the given port for the
informational-only field `<name>`.

Disable redirection and TLS termination altogether by specifying
`<name>#<port>#true`. This can be beneficial if the workload itself handles TLS
on that port or if the information exposed on this port is non-sensitive.

The following example workload exposes a web service on port 8080 and metrics on
port 7890. The web server is exposed to a 3rd party end-user which wants to
verify the deployment, therefore it's still required that the server hands out
it certificate chain signed by the mesh CA certificate. The metrics should be
exposed via TCP without TLS.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      initContainers:
        - name: sidecar
          image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy@sha256:..."
          restartPolicy: Always
          volumeMounts:
            - name: contrast-tls-certs
              mountPath: /tls-config
          env:
            - name: EDG_INGRESS_PROXY_CONFIG
              value: "web#8080#false##metrics#7890#true"
          securityContext:
            privileged: true
            capabilities:
              add:
                - NET_ADMIN
      containers:
        - name: web-svc
          image: ghcr.io/edgelesssys/frontend:v1.2.3@...
          ports:
            - containerPort: 8080
              name: web
            - containerPort: 7890
              name: metrics
          volumeMounts:
            - name: contrast-tls-certs
              mountPath: /tls-config
      volumes:
        - name: contrast-tls-certs
          emptyDir: {}
```

### Egress

To be able to route the egress traffic of the workload through Envoy, the remote
endpoints' IP address and port must be configurable.

* Choose an IP address inside the `127.0.0.0/8` CIDR and a port not yet in use
by the pod.
* Configure the workload to connect to this IP address and port.
* Set `<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>`
as `EDG_EGRESS_PROXY_CONFIG`. Separate multiple entries with `##`. Choose any
string identifying the service on the given port as `<name>`.

This redirects the traffic over Envoy. The endpoint must present a valid
certificate chain which must be verifiable with the
[mesh CA certificate](../architecture/certificates.md#usage-of-the-different-certificates).
Furthermore, Envoy uses a certificate chain with the mesh certificate of the workload
and the intermediate CA certificate as the client certificate.

The following example workload has no ingress connections and two egress
connection to different microservices. The microservices are themselves part
of the confidential deployment. One is reachable under `billing-svc:8080` and
the other under `cart-svc:8080`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      initContainers:
        - name: sidecar
          image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy@sha256:..."
          restartPolicy: Always
          volumeMounts:
            - name: contrast-tls-certs
              mountPath: /tls-config
          env:
            - name: EDG_EGRESS_PROXY_CONFIG
              value: "billing#127.137.0.1:8081#billing-svc:8080##cart#127.137.0.2:8081#cart-svc:8080"
          securityContext:
            privileged: true
            capabilities:
              add:
                - NET_ADMIN
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
          volumeMounts:
            - name: contrast-tls-certs
              mountPath: /tls-config
      volumes:
        - name: contrast-tls-certs
          emptyDir: {}
```
