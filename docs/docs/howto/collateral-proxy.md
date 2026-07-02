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
Save the following manifest as `collateral-proxy.yml`, adjusting the namespace and `storageClassName` to fit your cluster.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: collateral-proxy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: collateral-proxy
  serviceName: collateral-proxy
  template:
    metadata:
      labels:
        app.kubernetes.io/name: collateral-proxy
    spec:
      containers:
        - args:
            - -addr=:80
            - -state-dir=/var/lib/collateral-proxy
          image: ghcr.io/edgelesssys/contrast/collateral-proxy:latest
          name: collateral-proxy
          ports:
            - containerPort: 80
              name: proxy
          readinessProbe:
            httpGet:
              path: /healthz
              port: 80
            periodSeconds: 5
          resources:
            limits:
              memory: 256Mi
            requests:
              memory: 256Mi
          volumeMounts:
            - mountPath: /var/lib/collateral-proxy
              name: state
  volumeClaimTemplates:
    - apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: state
        namespace: default
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: collateral-proxy
  namespace: default
spec:
  ports:
    - name: proxy
      port: 80
      targetPort: 80
  selector:
    app.kubernetes.io/name: collateral-proxy
```

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

## Observability

The proxy exposes Prometheus metrics on its `/metrics` endpoint and a liveness check on `/healthz`.
To inspect the metrics, query the endpoint from inside the cluster, for example:

```bash
kubectl exec statefulset/collateral-proxy -- wget -qO- http://127.0.0.1/metrics
```
