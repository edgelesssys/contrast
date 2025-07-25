# Service mesh

The Contrast service mesh secures the communication of the workload by
automatically wrapping the network traffic inside mutual TLS (mTLS) connections.
The verification of the endpoints in the connection establishment is based on
certificates that are part of the
[PKI of the Coordinator](../architecture/certificates.md).

The service mesh can be enabled on a per-workload basis by adding a service mesh
configuration to the workload's object annotations. During the
`contrast generate` step, the service mesh is added as a
[sidecar container](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/)
to all workloads which have a specified configuration. The service mesh
container first sets up `iptables` rules based on its configuration and then
starts [Envoy](https://www.envoyproxy.io/) for TLS origination and termination.

## Configuring the proxy

The service mesh container can be configured using the following object
annotations:

- `contrast.edgeless.systems/servicemesh-ingress` to configure ingress.
- `contrast.edgeless.systems/servicemesh-egress` to configure egress.
- `contrast.edgeless.systems/servicemesh-admin-interface-port` to configure the
  Envoy admin interface. If not specified, no admin interface will be started.

If you aren't using the automatic service mesh injection and want to configure
the service mesh manually, set the environment variables
`CONTRAST_INGRESS_PROXY_CONFIG`, `CONTRAST_EGRESS_PROXY_CONFIG` and
`CONTRAST_ADMIN_PORT` in the service mesh sidecar directly.

### Ingress

All TCP ingress traffic is routed over Envoy by default. Since we use
[TPROXY](https://docs.kernel.org/networking/tproxy.html), the destination
address remains the same throughout the packet handling.

Any incoming connection is required to present a client certificate signed by
the
[mesh CA certificate](../architecture/certificates.md#usage-of-the-different-certificates).
Envoy presents a certificate chain of the mesh certificate of the workload and
the intermediate CA certificate as the server certificate.

If the deployment contains workloads which should be reachable from outside the
Service Mesh, while still handing out the certificate chain, disable client
authentication by setting the annotation
`contrast.edgeless.systems/servicemesh-ingress` as `<name>#<port>#false`.
Separate multiple entries with `##`. You can choose any descriptive string
identifying the service on the given port for the `<name>` field, as it's only
informational.

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
  annotations:
    contrast.edgeless.systems/servicemesh-ingress: "web#8080#false##metrics#7890#true"
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: web-svc
          image: ghcr.io/edgelesssys/frontend:v1.2.3@...
          ports:
            - containerPort: 8080
              name: web
            - containerPort: 7890
              name: metrics
```

When invoking `contrast generate`, the resulting deployment will be injected
with the Contrast service mesh as an init container.

```yaml
# ...
initContainers:
  - env:
      - name: CONTRAST_INGRESS_PROXY_CONFIG
        value: "web#8080#false##metrics#7890#true"
    image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy:v1.7.0@sha256:84a19c43413f60b42216e6fd024886397c673e29f15ecba8f6b11e6a95eb0560"
    name: contrast-service-mesh
    restartPolicy: Always
    securityContext:
      capabilities:
        add:
          - NET_ADMIN
      privileged: true
    volumeMounts:
      - name: contrast-secrets
        mountPath: /contrast
```

Note, that changing the environment variables of the sidecar container directly
will only have an effect if the workload isn't configured to automatically
generate a service mesh component on `contrast generate`. Otherwise, the service
mesh sidecar container will be regenerated on every invocation of the command.

### Egress

To be able to route the egress traffic of the workload through Envoy, the remote
endpoints' IP address and port must be configurable.

- Choose an IP address inside the `127.0.0.0/8` CIDR and a port not yet in use
  by the pod.
- Configure the workload to connect to this IP address and port.
- Set
  `<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>`
  as the `contrast.edgeless.systems/servicemesh-egress` workload annotation.
  Separate multiple entries with `##`. Choose any string identifying the service
  on the given port as `<name>`.

This redirects the traffic over Envoy. The endpoint must present a valid
certificate chain which must be verifiable with the
[mesh CA certificate](../architecture/certificates.md#usage-of-the-different-certificates).
Furthermore, Envoy uses a certificate chain with the mesh certificate of the
workload and the intermediate CA certificate as the client certificate.

The following example workload has no ingress connections and two egress
connection to different microservices. The microservices are part of the
confidential deployment. One is reachable under `billing-svc:8080` and the other
under `cart-svc:8080`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  annotations:
    contrast.edgeless.systems/servicemesh-egress: "billing#127.137.0.1:8081#billing-svc:8080##cart#127.137.0.2:8081#cart-svc:8080"
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
```
