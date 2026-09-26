# Attested TLS

Remote attestation verifies the integrity of code and configuration deployed to a confidential VM (CVM).
The concept is covered in depth on the [remote attestation](../attestation/overview.md) page.
Contrast uses remote attestation in two ways:

1. The Coordinator verifies workload initializers, according to the permitted workloads in the manifest.
2. Workload owners and data owners verify the Coordinator, according to its reproducible reference values.

Most of the time, verifiers want to establish a secure channel to the attester in order to exchange further messages.
For example, the Coordinator hands out secrets to workloads or seed share owners over this secure channel.
Contrast uses a protocol on top of TLS 1.3, together referred to as _aTLS_, to establish such a channel and verify the attestation at the same time.

## Conceptual messages

A TLS connection is established between a client and a server.
Contrast's aTLS, on the other hand, can attest unidirectional (client to server or server to client) and even bidirectional.
To avoid confusion, we're going to refer to the part requesting an attestation document as the _initiator_ and the part producing the document as the _responder_.

The protocol starts with the initiator creating a random value called a nonce.
This nonce is intended to demonstrate freshness of the attestation evidence.
If the responder embeds this nonce into the evidence, the initiator knows that the evidence was created specifically for this connection.

Next, the initiator sends an attestation request, including the nonce, to the responder.
The responder creates a fresh asymmetric private key, self-signs a certificate with this key, and uses both to establish a TLS connection to the initiator.
It also computes a cryptographic hash over the nonce and the public key, and requests an attestation report using the hash as `REPORTDATA`.

The responder now sends the attestation report back to the initiator, over the TLS channel just established.
The initiator observes the public key used by the responder, and calculates the expected hash from nonce and public key.
Finally, the initiator verifies that the attestation evidence matches the reference values and that the `REPORTDATA` is set to the expected hash.

At this point, the initiator knows that the responder runs the expected software, because it verified the evidence.
It can also be certain that the other end of the TLS channel terminates at the responder, because it knows that the expected software generated the key.
The initiator also knows that the key was generated for this TLS connection only because of the nonce embedded in the attestation report.
Thus, the attestation evidence is cryptographically tied to the established TLS channel, and the initiator successfully authenticated the responder.

## TLS extensions

Contrast embeds the exchange of attestation requests and responses into the TLS handshake.
Since there is no standard for such an embedding yet, Contrast repurposes existing TLS extension points.
While these weren't originally designed for conveying attestation documents, this approach allows to use Go's TLS implementation without modification.

To illustrate the full protocol, we're going to discuss mutual attestation, where both the TLS client and the TLS server act as initiator and responder.
For single-sided attestation, the respective requests and responses are simply not sent.

When a client initiates a TLS connection, it starts the TLS handshake by sending a `ClientHello` message to the server.
The request for attestation is included as an [Application-Layer Protocol Negotiation (ALPN)](https://www.rfc-editor.org/rfc/rfc7301) next protocol choice.
The server parses the nonce from the protocol string and creates the TLS private key and the attestation report.
It embeds the report as an [X.509 certificate extension](https://www.rfc-editor.org/rfc/rfc5280#section-4.2) into the self-signed certificate and sends the `ServerHello` and `ServerCertificate` TLS messages.

Now the server sends a [`CertificateRequest`](https://www.rfc-editor.org/rfc/rfc5246#section-7.4.4) message to the client, embedding the nonce into the _Distinguished Name_ field.
The client parses the nonce from this field and creates the TLS private key and the attestation report.
Like the server, it embeds the report as a certificate extension and sends it back to the server with the `ClientCertificate` message.

The following diagram shows the relevant messages carrying attestation protocol information.
Some messages of the TLS 1.3 handshake that aren't relevant to attestation are omitted.

```mermaid
sequenceDiagram
    Client->>Server: ClientHello { ALPN: [nonce-1] }
    Server->>Client: ServerCertificate { Ext: [report(nonce-1, pubkey-server)] }
    Server->>Client: CertificateRequest { DN: [nonce-2] }
    Client->>Server: ClientCertificate { Ext: [report(nonce-2, pubkey-client)] }
```

## Intra-handshake.fail

You might have heard about the [Intra-handshake.fail] paper by Sardar et.al.
This section explains the results of that paper and the implications for Contrast.

### Results

The paper analyses several ways of binding attestation evidence to a TLS channel.
Among these is the binding explained above, where the nonce and the ephemeral public key are hashed into the report data field.
Next, the authors define a set of TLS session correlation goals, and show that none of the analysed binding schemes meet these goals.
This result isn't really surprising for the schemes that are using ephemeral keys: the TEE commits itself to a specific key, not to a specific negotiated session.
For these schemes, the session could be relayed by anybody possessing the committed private key.

This wouldn't be an issue if all TEEs were perfectly secure and interchangeable, because the report would guarantee that the key was generated within the TEE and can't leave it.
In practice, however, this assumption doesn't hold universally.
TEEs can be attacked physically, for example with [TEE.fail], and keys could be extracted.
Thus we can't rely on the report only to assure key security, and in extension aTLS security.

### Applicability to Contrast

The necessary condition for the paper's relay attack is a leak of the private key by any TEE that otherwise passes verification.
If we rule out implementation errors, the threat model of the CPU vendors suggests that a hardware attack is necessary for that to happen.
This means that we can remove the attack vector by somehow ensuring physical security of the host systems.

Server CPUs have stable identifiers that can be verified, so we could use them to query the actual location, ownership and physical security.
In an ideal world, [Platform Ownership Endorsement] would assure that the expected cloud provider operates a particular TEE and vouches for its security.
Since the concept is quite new and the mechanism isn't widely deployed yet, we can't rely on it as of today.
Instead, Contrast users can add a list of expected hardware identifiers to their manifests.
Reports issued by another, attacker-controlled TEE won't pass validation due to ID mismatch.
This reduces the circle of possible attackers to the hardware owners, which is the baseline we can expect: a physical attack can compromise the session keys directly, making any attempt to secure the connection moot, regardless of binding strategy.

[Intra-handshake.fail]: https://www.researchgate.net/publication/408219182_Intra-handshakefail_CVE-2026-33697_High-severity_CVE_in_Attested_TLS
[TEE.fail]: https://tee.fail
[Platform Ownership Endorsement]: https://www.intel.com/content/www/us/en/developer/articles/technical/software-security-guidance/technical-documentation/platform-ownership-endorsements.html
