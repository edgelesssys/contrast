# Certificate authority

The Coordinator acts as a certificate authority (CA) for the workloads defined
in the manifest. After a workload pod's attestation has been verified by the
Coordinator, it receives a mesh certificate and the mesh CA certificate. The
mesh certificate can be used for example in a TLS connection as the server or
client certificate to proof to the other party that the workload has been
verified by the Coordinator. The other party can verify the mesh certificate
with the mesh CA certificate. While the certificates can be used by the workload
developer in different ways, they're automatically used in Contrast's service
mesh to establish mTLS connections between workloads in the same deployment.

## Public key infrastructure

The Coordinator establishes a public key infrastructure (PKI) for all workloads
contained in the manifest. The Coordinator holds three certificates: the root CA
certificate, the intermediate CA certificate, and the mesh CA certificate. The
root CA certificate is a long-lasting certificate and its private key signs the
intermediate CA certificate. The intermediate CA certificate and the mesh CA
certificate share the same private key. This intermediate private key is used to
sign the mesh certificates. Moreover, the intermediate private key and therefore
the intermediate CA certificate and the mesh CA certificate are rotated when
setting a new manifest.

![PKI certificate chain](../_media/contrast_pki.drawio.svg)

## Certificate rotation

Depending on the configuration of the first manifest, it allows the workload
owner to update the manifest and, therefore, the deployment. Workload owners and
data owners can be mutually untrusted parties. To protect against the workload
owner silently introducing malicious containers, the Coordinator rotates the
intermediate private key every time the manifest is updated and, therefore, the
intermediate CA certificate and mesh CA certificate. If the user doesn't trust
the workload owner, they use the mesh CA certificate obtained when they verified
the Coordinator and the manifest. This ensures that the user only connects to
workloads defined in the manifest they verified since only those workloads'
certificates are signed with this intermediate private key.

Similarly, the service mesh also uses the mesh CA certificate obtained when the
workload was started, so the workload only trusts endpoints that have been
verified by the Coordinator based on the same manifest. Consequently, a manifest
update requires a fresh rollout of the services in the service mesh.

## Usage of the different certificates

- The **root CA certificate** is returned when verifying the Coordinator. The
  data owner can use it to verify the mesh certificates of the workloads. This
  should only be used if the data owner trusts all future updates to the
  manifest and workloads. This is, for instance, the case when the workload
  owner is the same entity as the data owner.
- The **mesh CA certificate** is returned when verifying the Coordinator. The
  data owner can use it to verify the mesh certificates of the workloads. This
  certificate is bound to the manifest set when the Coordinator is verified. If
  the manifest is updated, the mesh CA certificate changes. New workloads will
  receive mesh certificates signed by the _new_ mesh CA certificate. The
  Coordinator with the new manifest needs to be verified to retrieve the new
  mesh CA certificate. The service mesh also uses the mesh CA certificate to
  verify the mesh certificates.
- The **intermediate CA certificate** links the root CA certificate to the mesh
  certificate so that the mesh certificate can be verified with the root CA
  certificate. It's part of the certificate chain handed out by endpoints in the
  service mesh.
- The **mesh certificate** is part of the certificate chain handed out by
  endpoints in the service mesh. During the startup of a pod, the Initializer
  requests a certificate from the Coordinator. This mesh certificate will be
  returned if the Coordinator successfully verifies the workload. The mesh
  certificate contains X.509 extensions with information from the workloads
  attestation document.
