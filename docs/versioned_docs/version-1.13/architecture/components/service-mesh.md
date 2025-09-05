# Service mesh

The Contrast service mesh secures the communication of the workload by automatically
wrapping the network traffic inside mutual TLS (mTLS) connections. The
verification of the endpoints in the connection establishment is based on
certificates that are part of the
[PKI of the Coordinator](#public-key-infrastructure).

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
The kata-agent only starts after this unit successfully runs and exits.
Therefore, the deny rule is in place before any containers can be started.

If you specify no service mesh annotation, or pass `--skip-service-mesh` to the CLI, the Initializer will be configured to remove the rule.

## Configuring the proxy

The service mesh container can be configured using the following object annotations:

- `contrast.edgeless.systems/servicemesh-ingress` to configure ingress.
- `contrast.edgeless.systems/servicemesh-egress` to configure egress.
- `contrast.edgeless.systems/servicemesh-admin-interface-port` to configure the Envoy
  admin interface. If not specified, no admin interface will be started.

Adding any of the ingress or egress annotations instructs the Contrast CLI to inject a service mesh sidecar container.
The sidecar is configured with the environment variables `CONTRAST_INGRESS_PROXY_CONFIG`, `CONTRAST_EGRESS_PROXY_CONFIG` and `CONTRAST_ADMIN_PORT`, which are set to their respective annotation's value.
After policy generation, the annotations themselves aren't interpreted by the runtime.

### Ingress

The service mesh ingress rule redirects all incoming TCP traffic transparently to the Envoy proxy.
The rule is activated if either the ingress or the egress annotation is present on the pod.
If the ingress annotation value is empty, incoming connections must present a client certificate signed by the [mesh CA certificate](#summary-of-certificate-roles).
Envoy presents a certificate chain of the mesh certificate of the workload and the intermediate CA certificate.

Exemptions from the mTLS requirement can be specified in the annotation value, by passing triples of the form `<name>#<port>#<disable-TLS>`,  separated by `##`.
If the deployment contains workloads which should be reachable from outside the
Service Mesh, while still handing out the certificate chain, disable client
authentication by setting the annotation `contrast.edgeless.systems/servicemesh-ingress` as
`<name>#<port>#false`.
You can choose any descriptive string identifying the service on the given port for the `<name>` field, as it's only informational.

Disable redirection and TLS termination altogether by specifying
`<name>#<port>#true`. This can be beneficial if the workload itself handles TLS
on that port or if the information exposed on this port is non-sensitive.

Setting the ingress annotation to the fixed string `DISABLED` will disable the rule altogether.
This is useful when egress rules are desired, but ingress rules aren't.

The following example workload exposes a web service on port 8080 and metrics on port 7890.
The web server is exposed to a 3rd party end-user who wants to verify the deployment, therefore it's still required that the server hands out its certificate chain.
The metrics endpoint should be exposed directly, without TLS.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-ingress: "web#8080#false##metrics#7890#true"
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: web-svc
          image: ghcr.io/edgelesssys/frontend:v1.2.3@sha256:...
          ports:
            - containerPort: 8080
              name: web
            - containerPort: 7890
              name: metrics
```

When invoking `contrast generate`, the resulting deployment will be injected with the
Contrast service mesh as an init container:

```yaml
# ...
spec:
    runtimeClassName: contrast-cc
    containers:
      - name: web-svc
        image: ghcr.io/edgelesssys/frontend:v1.2.3@sha256:...
        ports:
          - containerPort: 8080
            name: web
          - containerPort: 7890
            name: metrics
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
[mesh CA certificate](#summary-of-certificate-roles).
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
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-egress: "billing#127.137.0.1:8081#billing-svc:8080##cart#127.137.0.2:8081#cart-svc:8080"
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@sha256:...
```

Setting the `contrast.edgeless.systems/servicemesh-egress` annotation without a value will result in an error during `contrast generate`.

## Public key Infrastructure

The Coordinator establishes a public key infrastructure (PKI) for all workloads defined in the manifest. It holds three types of certificates:

- **Root CA certificate**: A long-lived certificate used to sign the intermediate CA certificate.
- **Intermediate CA certificate**: Shares a private key with the mesh CA certificate and is signed by the root CA. This private key is used to sign mesh certificates.
- **Mesh CA certificate**: Used to sign workload-specific mesh certificates.

![PKI certificate chain](../../_media/contrast_pki.drawio.svg)

### Certificate issuance

Once a workload pod’s attestation is successfully verified by the Coordinator, it receives:

- A **mesh certificate**
- The **mesh CA certificate**

The mesh certificate includes X.509 extensions based on the workload’s attestation document and can be used as a client or server certificate in TLS connections. It proves to the remote party that the workload was verified by the Coordinator. The remote party can verify the mesh certificate using the mesh CA certificate.

While developers may use these certificates independently, they're also automatically used by Contrast’s service mesh for secure communication.

### Certificate rotation

Every time the manifest is updated, the Coordinator rotates the intermediate private key. As a result, both the intermediate CA certificate and the mesh CA certificate are renewed.

This mechanism protects against scenarios where a workload owner introduces unauthorized containers after verification. If a user doesn't trust the workload owner, they should only trust mesh certificates signed by the mesh CA certificate obtained during their own verification process.

Similarly, the service mesh uses the mesh CA certificate issued when the workload was verified. Any change to the manifest requires a new rollout of the services, as the mesh CA certificate will change.

### Service mesh integration

The service mesh relies on the mesh certificates to establish mutual TLS (mTLS) connections between workloads.

- During pod startup, the Initializer requests a mesh certificate from the Coordinator.
- If attestation is successful, the Coordinator returns a mesh certificate and the mesh CA certificate.
- These certificates are used to authenticate and authorize communication within the service mesh.

Only workloads that have been verified based on the current manifest and signed by the corresponding mesh CA certificate are trusted by other services in the mesh.

### Summary of certificate roles

- **Root CA Certificate**
  Returned during Coordinator verification. Can be used to verify mesh certificates if the data owner fully trusts all future manifests and updates.

- **Intermediate CA Certificate**
  Links the root CA to the mesh CA. Included in certificate chains for validation purposes.

- **Mesh CA Certificate**
  Used to verify mesh certificates. Bound to a specific manifest version and changes when the manifest is updated.

- **Mesh Certificate**
  Issued to workloads after successful attestation. Contains metadata from the attestation document and is used in mTLS communication within the service mesh.
