# Communicate with workloads

<!--TODO(katexochen): This section needs rework.
    The step was part of the old tutorial but this doesn't make much sense as a stand alone how-to. -->

This section explains how to use the Contrast service mesh to communicate with your application workloads.

## Applicability

This step is optional and only relevant if you have configured Contrast to use the service mesh PKI for incoming connections to your application.

## Prerequisites

1. A running Contrast deployment
2. [Configure TLS](./TLS-configuration.md)

## How-to

You can securely connect to the workloads using the Coordinator's `mesh-ca.pem` as a trusted CA certificate.
First, expose the service on a public IP address via a LoadBalancer service:

```sh
kubectl patch svc ${MY_SERVICE} -p '{"spec": {"type": "LoadBalancer"}}'
kubectl wait --timeout=30s --for=jsonpath='{.status.loadBalancer.ingress}' service/${MY_SERVICE}
lbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo $lbip
```

Using `openssl`, the certificate of the service can be verified with the `mesh-ca.pem`:

```sh
openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null
```

:::info[Subject alternative names and LoadBalancer IP]

By default, mesh certificates are issued with a wildcard DNS entry.
Here, the service is accessed via its LoadBalancer IP address.
Tools like curl check the certificate for IP entries in the subject alternative name (SAN) field, which aren't present.
For example, attempting to connect with curl and the mesh CA certificate will throw the following error:

```sh
$ curl --cacert ./verify/mesh-ca.pem "https://${frontendIP}:443"
curl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'
```

Check out the [SANs section in the manifest reference](../../architecture/components/manifest.md#policies-sans) to learn how to add IP addresses to the SANs of your workload certificates.

:::
