# RFC 001: Service Mesh

Applications inside Confidential Containers should be able to talk to each other
confidentially without the need to adapt the source code of the applications.

## The Problem

Configuring the CA and client certificates inside the applications is tedious,
since it involves developers changing their code in multiple places. This also
breaks the lift and shift promise. Therefore, we can only expect the user to
make slight changes to their deployments.

## Solution

We will deploy a sidecar container[1] which consumes the CA and client
certificates. It can establish mTLS connections to other applications enrolled
in the mesh by connecting to their sidecar proxies.

All ingress and egress traffic should be routed over the proxy. The proxy should
route packets to the original destination IP and port. Additionally, the proxy
must be configured on which ingress endpoints to enforce client authentication.

[1] <https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/>

The problem left is how to route the applications traffic over the proxy. We
propose 2 routing solutions and 2 proxy solutions.

### Routing Solution 1: Manually map ingress and egress

This solution shifts the
`all ingress and egress traffic should be routed over the proxy` requirement to
the user.

Additionally, this solutions requires that the service endpoints are
configurable inside the deployments. Since this is the case for both emojivoto
as well as Google's microservice demo, this is a reasonable requirement.

For ingress traffic, we define a port mapping from the proxy to the application.
All traffic that target the proxy on that port will be forwarded to the other
port. We also need to protect the application from being talked to directly via
the port it exposes. To achieve that, we block all incoming traffic to the
application via `iptables`.

For egress traffic, we configure a port and an endpoint. The proxy will listen
locally on the port and forward all traffic to the specified endpoint. We set
the endpoint in the application setting to `localhost:port`.

### Routing Solution 2: `iptables` based re-routing

With this solution we take care of the correct routing for the user and have no
requirements regarding configuration of endpoints.

One example of `iptables` based routing is Istio [1] [2] [3]. In contrast to
Istio, we don't need a way to configure anything dynamically, since we don't
have the concept of virtual services and also our certificates are wildcard
certificates per default.

[1] <https://github.com/istio/istio/wiki/Understanding-IPTables-snapshot>

[2]
<https://tetrate.io/blog/traffic-types-and-iptables-rules-in-istio-sidecar-explained/>

[3]
<https://jimmysongio.medium.com/sidecar-injection-transparent-traffic-hijacking-and-routing-process-in-istio-explained-in-detail-d53e244e0348>

### Proxy Solution 1: Custom implemented TPROXY

TPROXY [1] is a kernel feature to allow applications to proxy traffic without
changing the actual packets, for example when re-routing them via NAT.

The proxy can implement custom user-space logic to handle traffic and route the
traffic to the original destination (see a simple Go example [2]).

We likely re-implement parts of Envoy (see below), but have more flexibility
regarding additional verification, for example in case we decide to also use
custom client certificate extensions.

[1] <https://www.kernel.org/doc/Documentation/networking/tproxy.txt>

[2]
<https://github.com/KatelynHaworth/go-tproxy/blob/master/example/tproxy_example.go>

### Proxy Solution 2: Envoy

Envoy is a L3/4/7 proxy used by Istio and Cilium. In combination with either
`iptables REDIRECT` or TPROXY it can be used to handle TLS origination and
termination [1]. The routing will be done via the original destination filter
[2]. For TLS origination we wrap all outgoing connections in TLS since we can't
rely on DNS to be secure. Istio uses "secure naming" [3] to at least protect
HTTP/HTTPS traffic from DNS spoofing, but r.g., raw TCP or UDP traffic isn't
secured.

[1]
<https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/ssl.html#tls>

[2]
<https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/listener_filters/original_dst_filter>

[3] <https://istio.io/latest/docs/concepts/security/#secure-naming>

## General questions

- Which traffic do we want to secure? HTTP/S, TCP, UDP, ICMP? Is TLS even the
  correct layer for this?

Since TCP service meshes are ubiquitously used, only supporting TCP for now is
fine.

- Do we allow workloads to talk to the internet by default? Otherwise we can
  wrap all egress traffic in mTLS.

For egress a secure by default option would be nice, but is hard to achieve.
This can be implemented in a next step.

- Do we want to use any custom extensions in the client certificates in the
  future?

No, for now we don't use any certificate extensions which bind the certificate
to the workload.

## Way forward

In Kubernetes the general architecture will be to use a sidecar container which
includes an Envoy proxy and a small Go or Bash program to configure routes and
setup and configure Envoy.

### Step 1: Egress

The routing works on layer 3. The workload owner configures the workload's
service endpoints to point to a unique local IP out of the 127.0.0.1/8 CIDR. The
workload owner configures the proxy to listen on each of those addresses and map
it to a remote service domain.

If possible, we don't want to touch the port of the packets so that we can
transparently proxy all ports of a service.

Note that this isn't secure by default. If the user doesn't configure the
endpoints in their application, traffic is send out unencrypted and without
authentication.

<img src="./assets/001-egress.svg">

### Step 2: Ingress

For ingress traffic we deploy `iptables` rules which redirect all traffic to
Envoy via TPROXY `iptables` rules. After Envoy has terminated the TLS
connection, it sends out the traffic again to the workload. The routing is
similar to what Istio does [1].

The user can configure an allow list of ports which shouldn't be redirected to
Envoy. Also traffic originating from the UID the proxy is started with, isn't
redirected. Since by default all traffic is routed to Envoy, the workload's
ingress endpoint are secure by default.

<img src="./assets/001-ingress.svg">

### Step 3: Secure by default egress

Ideally, we also want to also have secure by default egress. But this comes with
additional tradeoffs. If we assume that the workload does _NOT_ talk to any
other endpoints outside of the service mesh, then we can redirect all traffic
through the proxy. Since we can't assume this to be true for all workloads, we
still need the explicit configuration method described above.

Since we need to allow DNS for Kubernetes service lookup, we can only redirect
all TCP traffic via the proxy.

### Optional: Egress capturing via DNS

If we want to allow additional endpoints, we also need to touch the pod's DNS
resolution. An easy way would be to resolve the allow-listed entries to either
directly the correct endpoint or to a special IP of the proxy. This required the
application to not implement basic DNS (over UDP) and not DNS-over-HTTPS,
DNS-over-QUIC, or similar.

### Outlook

Especially for ingress but also for egress as described in step 3, we must
ensure that the sidecar/init container runs before the workloads receives
traffic. Otherwise, it might be that the `iptables` rules aren't configured yet
and the traffic is send without TLS and without client verification.
