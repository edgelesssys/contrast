# Service mesh

The Contrast service mesh secures the communication of the workload by automatically
wrapping the network traffic inside mutual TLS (mTLS) connections. The
verification of the endpoints in the connection establishment is based on
certificates that are part of the
[PKI of the Coordinator](#certificate-authority).

The service mesh can be enabled on a per-workload basis by adding a service mesh
configuration to the workload's object annotations. During the `contrast generate`
step, the service mesh is added as a [sidecar
container](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/) to
all workloads which have a specified configuration. The service mesh container first
sets up `iptables` rules based on its configuration and then starts
[Envoy](https://www.envoyproxy.io/) for TLS origination and termination.

## Service mesh startup enforcement

Since Contrast doesn't yet enforce the order in which the containers are started
(see [Limitations](../features-limitations.md)), we deny all incoming connections
until the service mesh is fully configured.
A systemd unit inside the podVM creates this deny rule.
The kata-agent systemd unit requires that this unit successfully runs and exits,
before itself it can be started.
Therefore, the deny rule is in place before any containers can be started.

If the user specifies no service mesh annotations, the Initializer takes care
of removing the deny rule.

## Configuring the proxy

The service mesh container can be configured using the following object annotations:

- `contrast.edgeless.systems/servicemesh-ingress` to configure ingress.
- `contrast.edgeless.systems/servicemesh-egress` to configure egress.
- `contrast.edgeless.systems/servicemesh-admin-interface-port` to configure the Envoy
  admin interface. If not specified, no admin interface will be started.

If you aren't using the automatic service mesh injection and want to configure the
service mesh manually, set the environment variables `CONTRAST_INGRESS_PROXY_CONFIG`,
`CONTRAST_EGRESS_PROXY_CONFIG` and `CONTRAST_ADMIN_PORT` in the service mesh sidecar directly.

### Ingress

All TCP ingress traffic is routed over Envoy by default. Since we use
[TPROXY](https://docs.kernel.org/networking/tproxy.html), the destination address
remains the same throughout the packet handling.

Any incoming connection is required to present a client certificate signed by the
[mesh CA certificate](#usage-of-the-different-certificates).
Envoy presents a certificate chain of the mesh
certificate of the workload and the intermediate CA certificate as the server certificate.

If the deployment contains workloads which should be reachable from outside the
Service Mesh, while still handing out the certificate chain, disable client
authentication by setting the annotation `contrast.edgeless.systems/servicemesh-ingress` as
`<name>#<port>#false`. Separate multiple entries with `##`. You can choose any
descriptive string identifying the service on the given port for the `<name>` field,
as it's only informational.

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

When invoking `contrast generate`, the resulting deployment will be injected with the
Contrast service mesh as an init container.

```yaml
# ...
initContainers:
  - env:
      - name: CONTRAST_INGRESS_PROXY_CONFIG
        value: "web#8080#false##metrics#7890#true"
    image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy:latest"
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

Note, that changing the environment variables of the sidecar container directly will
only have an effect if the workload isn't configured to automatically generate a
service mesh component on `contrast generate`. Otherwise, the service mesh sidecar
container will be regenerated on every invocation of the command.

### Egress

To be able to route the egress traffic of the workload through Envoy, the remote
endpoints' IP address and port must be configurable.

- Choose an IP address inside the `127.0.0.0/8` CIDR and a port not yet in use
  by the pod.
- Configure the workload to connect to this IP address and port.
- Set `<name>#<chosen IP>:<chosen port>#<original-hostname-or-ip>:<original-port>`
  as the `contrast.edgeless.systems/servicemesh-egress` workload annotation. Separate multiple
  entries with `##`. Choose any string identifying the service on the given port as
  `<name>`.

This redirects the traffic over Envoy. The endpoint must present a valid
certificate chain which must be verifiable with the
[mesh CA certificate](#usage-of-the-different-certificates).
Furthermore, Envoy uses a certificate chain with the mesh certificate of the workload
and the intermediate CA certificate as the client certificate.

The following example workload has no ingress connections and two egress
connection to different microservices. The microservices are part
of the confidential deployment. One is reachable under `billing-svc:8080` and
the other under `cart-svc:8080`.

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

## Certificate authority

The Coordinator acts as a certificate authority (CA) for the workloads
defined in the manifest.
After a workload pod's attestation has been verified by the Coordinator,
it receives a mesh certificate and the mesh CA certificate.
The mesh certificate can be used for example in a TLS connection as the server or
client certificate to proof to the other party that the workload has been
verified by the Coordinator. The other party can verify the mesh certificate
with the mesh CA certificate. While the certificates can be used by the workload
developer in different ways, they're automatically used in Contrast's service
mesh to establish mTLS connections between workloads in the same deployment.

### Public key infrastructure

The Coordinator establishes a public key infrastructure (PKI) for all workloads
contained in the manifest. The Coordinator holds three certificates: the root CA
certificate, the intermediate CA certificate, and the mesh CA certificate.
The root CA certificate is a long-lasting certificate and its private key signs
the intermediate CA certificate. The intermediate CA certificate and the mesh CA
certificate share the same private key. This intermediate private key is used
to sign the mesh certificates. Moreover, the intermediate private key and
therefore the intermediate CA certificate and the mesh CA certificate are
rotated when setting a new manifest.

![PKI certificate chain](../../_media/contrast_pki.drawio.svg)

### Certificate rotation

Depending on the configuration of the first manifest, it allows the workload
owner to update the manifest and, therefore, the deployment.
Workload owners and data owners can be mutually untrusted parties.
To protect against the workload owner silently introducing malicious containers,
the Coordinator rotates the intermediate private key every time the manifest is
updated and, therefore, the
intermediate CA certificate and mesh CA certificate. If the user doesn't
trust the workload owner, they use the mesh CA certificate obtained when they
verified the Coordinator and the manifest. This ensures that the user only
connects to workloads defined in the manifest they verified since only those
workloads' certificates are signed with this intermediate private key.

Similarly, the service mesh also uses the mesh CA certificate obtained when the
workload was started, so the workload only trusts endpoints that have been
verified by the Coordinator based on the same manifest. Consequently, a
manifest update requires a fresh rollout of the services in the service mesh.

### Usage of the different certificates

- The **root CA certificate** is returned when verifying the Coordinator.
  The data owner can use it to verify the mesh certificates of the workloads.
  This should only be used if the data owner trusts all future updates to the
  manifest and workloads. This is, for instance, the case when the workload owner is
  the same entity as the data owner.
- The **mesh CA certificate** is returned when verifying the Coordinator.
  The data owner can use it to verify the mesh certificates of the workloads.
  This certificate is bound to the manifest set when the Coordinator is verified.
  If the manifest is updated, the mesh CA certificate changes.
  New workloads will receive mesh certificates signed by the _new_ mesh CA certificate.
  The Coordinator with the new manifest needs to be verified to retrieve the new mesh CA certificate.
  The service mesh also uses the mesh CA certificate to verify the mesh certificates.
- The **intermediate CA certificate** links the root CA certificate to the
  mesh certificate so that the mesh certificate can be verified with the root CA
  certificate. It's part of the certificate chain handed out by
  endpoints in the service mesh.
- The **mesh certificate** is part of the certificate chain handed out by
  endpoints in the service mesh. During the startup of a pod, the Initializer
  requests a certificate from the Coordinator. This mesh certificate will be returned if the Coordinator successfully
  verifies the workload. The mesh certificate
  contains X.509 extensions with information from the workloads attestation
  document.
