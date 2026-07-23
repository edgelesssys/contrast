# Caching attestation collateral

This section describes how to deploy and use the Contrast collateral proxy, an in-cluster cache for the attestation collateral that Contrast fetches from hardware vendors.

## Applicability

To verify an attestation report, Contrast retrieves *collateral* from external vendor endpoints:
VCEK/VLEK certificates and CRLs from the [AMD Key Distribution Service (KDS)](../architecture/attestation/amd-details.md#attestation-report),
PCK certificates, TCB info, and QE identities from the Intel Provisioning Certification Service (PCS), and RIM files from NVIDIA.
Both the [Coordinator](../architecture/components/coordinator.md) and the [Initializer](../architecture/components/initializer.md) fetch this collateral
before attesting a peer.

Unfortunately, these endpoints can occasionally become unavailable, for example due to rate limiting on the vendors' side, or through transient networking errors.
To counter this, a `collateral-proxy` can be deployed as shown below and used by all Contrast coordinators and initializers.

## Prerequisites

A running Contrast deployment.

## How-To

Using the proxy requires deploying the proxy once per cluster, then pointing your Contrast components at it when generating your deployment.

### Deploy the proxy

The proxy is deployed once per cluster and shared by all Contrast deployments.
Download the latest version's deployment:

```sh
curl -fLO https://github.com/edgelesssys/collateral-proxy/releases/latest/download/collateral-proxy.yml
```

Adjust the namespace and `storageClassName` to fit your cluster.
Apply it:

```bash
kubectl apply -f collateral-proxy.yml
```

The proxy persists its cache on the `state` volume (`1Gi` by default), allowing the cache to survive restarts.
The proxy itself runs as an ordinary pod, not as a confidential workload.
It only ever handles signed, publicly available collateral.

### Route components through the proxy

Pass the proxy's in-cluster base URL to `contrast generate` with the `--collateral-proxy` flag, then apply your deployment:

```bash
contrast generate --collateral-proxy http://collateral-proxy.default.svc resources/
kubectl apply -f resources/
```

With this flag, coordinators and initializers route their AMD KDS and Intel PCS collateral fetches through the proxy instead of contacting the vendor endpoints directly.
Without the flag, components fetch collateral directly and the proxy isn't required.

The proxy is a soft dependency: if a component can't reach it, the component logs a warning and falls back to fetching directly from the vendor endpoint, then retries the proxy after a short cooldown.

### Internal request flow

Internally, the proxy fetches and caches based on the following logic.

1. Look up the reconstructed upstream URL in the cache.
2. Fresh cache entry exists: return the cached response directly.
3. Stale cache entry exists: fetch the upstream.
   - On success, the response is returned to the caller.
   - On upstream failure, return stale entry as fallback.
4. No cache entry exists: fetch the upstream.
   - On success, the response is returned to the caller.
   - On upstream failure, return `502 Bad Gateway`.

Relevant response headers are forwarded to the client.
Only `200 OK` responses from upstream are cached.
The cache duration follows the `max-age` or `nextUpdate` fields if provided by upstream, otherwise defaults to 1 hour.

### Observability

The proxy exposes Prometheus metrics on its `/metrics` endpoint and a liveness check on `/healthz`.
To inspect the metrics, query the endpoint from inside the cluster, for example:

```bash
kubectl exec statefulset/collateral-proxy -- wget -qO- http://127.0.0.1/metrics
```

Alternatively, configure port forwarding as you usually would.
No special considerations are required, since the proxy doesn't run in a Confidential Containers pod.

## Security considerations

The proxy is served over plain HTTP, and its in-cluster traffic isn't encrypted or authenticated at the transport layer.
This is safe by design, because Contrast never trusts the proxy, or the connection to it, to vouch for the collateral it returns.

Instead, every piece of collateral is authenticated cryptographically after it's fetched:

- Each Contrast component ships with the hardware vendors' root certificates embedded in its binary, namely the AMD ARK and ASK certificates for SEV-SNP, and the Intel SGX Provisioning Certification root CA for TDX.
- The underlying libraries used in Contrast verify the full certificate chain of the fetched collateral against these embedded roots, and check the attestation report signature against the leaf key.

Because this verification is anchored in keys baked into the component at build time rather than in the connection to the proxy, transport encryption adds no additional security, and conversely, lack of transport encryption doesn't detract from security.

:::warning
**A compromised proxy, or an attacker on the network path to it, can't forge collateral that passes verification.**

The worst it can do is withhold collateral or serve a stale copy, which either fails verification or, if the proxy becomes unreachable, triggers the fallback to direct vendor fetching described above.

Additionally, the proxy only ever handles signed, publicly available collateral.
It never sees secrets, attestation reports, or workload data, so there's nothing confidential to protect on the wire.
:::
