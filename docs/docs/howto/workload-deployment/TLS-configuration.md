# Configure TLS

Contrast supports communication via a Contrast-specific PKI with certifcates being issues based on successful attestation.

## Applicability

This step is optional, but it is highly recommended to consider this step when deploying your application with Contrast.
If nothing is configured, all incoming traffic to workloads is required to present a valid client certificate.
By default, no outgoing traffic is secured.

## Prerequisites

1. [Set up cluster](.)
2. [Deploy runtime](.)
3. [Prepare deployment files](.)

## How-to

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
