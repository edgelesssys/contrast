# RFC 011: Registry Authentication

## Background

### Registry credentials

Most container registries used in enterprise environments require authentication to pull images.
There are several ways in which these credentials can be supplied to the container runtime:

1. Image pull secret referenced directly in the pod spec: <https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/>.
2. Image pull secret attached to the service account: <https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account>. This indirectly populates the `imagePullSecrets` field of all pod specs using this service account.
3. Global credentials in the container runtime configuration: <https://github.com/containerd/containerd/blob/5bcf77a/docs/cri/registry.md?plain=1#L37-L40>.
4. Dedicated image credential providers: <https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/>.

Image pull secrets come in [two formats](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets).
The `dockercfg` format has been [removed](https://github.com/docker/cli/pull/2504) by Docker itself, but this wasn't too long ago and it's likely still widely used.

### Registry certificates

Internal registries often use certificates that don't chain back to a web PKI root CA, but an internal one.
Since Kubernetes doesn't have an API for specifying trusted registry CAs, such CA certificates need to be [configured at the container runtime](https://github.com/containerd/containerd/blob/5bcf77a55038ad658c57fdecc48af54935a0d42f/docs/cri/config.md?plain=1#L744).

### Proxies

Many corporate networks require HTTP connections to the public internet to go through a proxy.
The de-facto standard for specifying such proxies are the environment variables `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`.
If Contrast is running in such an environment, it needs to respect these variables in order to pull images from within the guest.

## Requirements

1. Users must be able to guest-pull images from registries that require authentication.
2. Users must be able to guest-pull images from registries that don't participate in web PKI.
3. Users must be able to guest-pull images through HTTP proxies.

## Design

The overall idea of this design is to supply an image puller configuration to the VM that's _not measured_.
Reason for this is that the image pull configuration is a workload owner secret that shouldn't be visible to (unauthenticated) verifiers - see [Alternatives considered](#alternatives-considered).

### Changes to the `imagepuller`

We add support for an optional configuration file to the `imagepuller`, with the following structure:

```go
import "github.com/google/go-containerregistry/pkg/authn"

type InsecureConfig struct {
    Registries map[string]Registry // mapping of domain name patterns to registry configuration
    ExtraEnv map[string]string
}

type Registry struct {
    authn.AuthConfig
    CACerts string // concatenated, PEM-encoded CA certificates (system certs if nil)
    InsecureSkipVerify bool
}
```

On startup, the `imagepuller` reads the config file from `/run/insecure-cfg/imagepuller.TBD`.
Each key-value pair in `ExtraEnv` is added to the `imagepuller`s own environment.
The `Registries` are stored in a field.

When a `PullImage` request comes in, the `imagepuller` extracts the registry part from the image reference.
It then matches the incoming registry to the keys of `Registries`, using the [algorithm in the appendix](#registry-name-matching).

If the `Registry` contains an [`authn.AuthConfig`], it's wrapped with [`authn.FromConfig`] and passed as an option [`remote.WithAuth`].
The imagepuller constructs a new [`tls.Config`].
If `CACerts` isn't empty, it replaces the pool with a new pool using those certificates.
If `InsecureSkipVerify` is set, the same is set on the [`tls.Config`].
The [`tls.Config`] and [`http.ProxyFromEnvironment`] are used to construct an [`http.Transport`] that's passed as an option [`remote.WithTransport`].

[`tls.Config`]: https://pkg.go.dev/crypto/tls#Config
[`http.ProxyFromEnvironment`]: https://pkg.go.dev/net/http#ProxyFromEnvironment
[`http.Transport`]: https://pkg.go.dev/net/http#Transport
[`authn.AuthConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#AuthConfig
[`authn.FromConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#FromConfig
[`remote.WithAuth`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithAuth
[`remote.WithTransport`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithTransport

#### Example 1

In this scenario, there's an internal registry at `registry.corp` for workload images, but `ghcr.io` is used for Contrast images.
The internal registry is served from within the corporate firewall, while access to external registries is mediated by a proxy.
Other registries are allowed, as long as they don't require authentication.

```toml
[extra-env]
HTTP_PROXY = "https://proxy.corp"
HTTPS_PROXY = "https://proxy.corp"
NO_PROXY = ".corp"

[registries."registry.corp."]
ca-certs = '''
-----BEGIN CERTIFICATE-----
MIIBezCCASGgAwIBAgIUUugBbePTzyVApU4DLSMmHnXXjcwwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAwwIWW91ck5hbWUwHhcNMjUxMDIxMTUwNDI0WhcNMjYxMDIxMTUw
NDI0WjATMREwDwYDVQQDDAhZb3VyTmFtZTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABIgsA5IEeiBq6jDpH2ttxrI96beeOqa+EpGqmznQmzpFkPEpLWMUt21Ien71
rxdeFC7ySuuu95VPjSvO7EUM9qyjUzBRMB0GA1UdDgQWBBTVnuI2o36Mrja3RvwE
82lWg2m19zAfBgNVHSMEGDAWgBTVnuI2o36Mrja3RvwE82lWg2m19zAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMCA0gAMEUCIGmEkl8jxjxqyAxs3QoAXeIx++Bz
Zm9dwbeTbrKysrGXAiEA8ce6iyJUCZCZVVJs/HDLcPbOKc2EPZvdcGGjIlGXulo=
-----END CERTIFICATE-----
'''

[registries."ghcr.io."]
auth = "YnVyZ2VyZGV2OnRoaXNpc25vdG15cGFzc3dvcmQ="
```

#### Example 2

In this scenario, all container images are served from a public registry.
However, the image owner wants to make sure that the traffic can't be intercepted by rogue CAs.
Other registries are strictly forbidden.

```toml
[registries."very-secure.registri.es"]
auth = "bmljZTp0cnk="
ca-certs = '''
Root CA 1
-----BEGIN CERTIFICATE-----
MIIBfDCCASGgAwIBAgIUU5G42y9bIh8+AU38qVOmKocc0CwwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAwwIWW91ck5hbWUwHhcNMjUxMDIxMTUyNzEwWhcNMjYxMDIxMTUy
NzEwWjATMREwDwYDVQQDDAhZb3VyTmFtZTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABOJlyBb/sHBmHRncTqk4lm6hBkBYlZGcScXfl/IuAVVIo4zCGBzCmvc7jYc2
+gyVp+wxuvm7NRza4e1QOfJfrxOjUzBRMB0GA1UdDgQWBBTRE8qju+GIWzr5xCik
MdBJFOd1lzAfBgNVHSMEGDAWgBTRE8qju+GIWzr5xCikMdBJFOd1lzAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMCA0kAMEYCIQCn+fVmAzB8HOakKGLx6oXF0WP0
GJibphhjfHPdNWEDdQIhAN3KFNWIYtE35+/rZb5I+oVKnqKS8igdIU9lXmpOps1j
-----END CERTIFICATE-----
Root CA 2
-----BEGIN CERTIFICATE-----
MIIBezCCASGgAwIBAgIUUugBbePTzyVApU4DLSMmHnXXjcwwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAwwIWW91ck5hbWUwHhcNMjUxMDIxMTUwNDI0WhcNMjYxMDIxMTUw
NDI0WjATMREwDwYDVQQDDAhZb3VyTmFtZTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABIgsA5IEeiBq6jDpH2ttxrI96beeOqa+EpGqmznQmzpFkPEpLWMUt21Ien71
rxdeFC7ySuuu95VPjSvO7EUM9qyjUzBRMB0GA1UdDgQWBBTVnuI2o36Mrja3RvwE
82lWg2m19zAfBgNVHSMEGDAWgBTVnuI2o36Mrja3RvwE82lWg2m19zAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMCA0gAMEUCIGmEkl8jxjxqyAxs3QoAXeIx++Bz
Zm9dwbeTbrKysrGXAiEA8ce6iyJUCZCZVVJs/HDLcPbOKc2EPZvdcGGjIlGXulo=
-----END CERTIFICATE-----
'''

[registries."."]
ca-certs = "no PEM here means no CA certificates"
```

#### Example 3

In this scenario, there's an HTTP-only registry deployed into the cluster for ease of use.
Transport security for this internal registry isn't important to the operators.
Other registries should be used anonymously, but with TLS.

```toml
[registries."registry.default.svc.cluster.local."]
insecure-skip-verify = true
```

### Example 4

This scenario is equivalent to the prior behavior: web PKI and no authentication.

```toml
```

### Changes to the node-installer

The node-installer gets another optional volume mount, similar to the [target config](https://github.com/edgelesssys/contrast/blob/6ef858031759966dfd3cdeda2b4570bed45fdcda/internal/kuberesource/parts.go#L126-L131) but using a secret.
The secret contains one key, `imagepuller.TBD`, containing a serialized version of `InsecureConfig` defined above.
This secret is created by the k8s administrator (or the workload owner) before applying the runtime.
If the node-installer finds a mounted secret, it writes the content into `/opt/edgeless/contrast-cc-*/etc/host-config/imagepuller.TBD`.

node-installer operations are intended to be idempotent.
In order to change the imagepuller configuration, the k8s administrator only needs to change the secret and restart the `DaemonSet`.

### Changes to the Kata runtime

During sandbox creation, the Kata runtime looks up the path to the imagepuller config in its own configuration.
The default is `/opt/edgeless/contrast-cc-*/etc/host-config/imagepuller.TBD`, but it can be overridden with the appropriate Kata annotation.
It packs the file into a device and attaches it to the VM.
This code should follow the initdata device provisioning logic very closely, but use a different magic identifier (TBD).

### Changes to the image

We add a new functionality to the initdata-processor:
After verifying initdata, the initdata-processor scans for the device introduced above.
The content of this device is copied over to `/run/insecure-cfg`, without any integrity checks.
The name is generic to allow for future use cases outside of image pulling.

### Security considerations

We need to be careful about the trust we put into configuration by the host.
For example, it would be a bad idea to allow switching off digest validation depending on such host configuration.
However, the fields in the imagepuller config aren't a risk to guest integrity, which is explained in the following subsections.

#### Wrong CA certificates

The CA certificates aren't required for image integrity - that's accomplished by pinning references.
Rather, the CA certificate option serves two purposes:

1. Allow connecting to a registry that isn't publicly trusted in the web PKI.
2. Restrict who can ostensibly intercept and log traffic for metadata analysis (for example, what images are loaded).

An attacker with k8s admin privileges could configure CA certificates to serve registry requests from unexpected endpoints.
They'd still need to serve the correct, pinned image, so they can only record traffic passively.
However, the image data is already exposed to the host (due to containerd's host pull), and so are the registry credentials.

The argument is very similar for `InsecureSkipVerify` and proxy environment variables.

#### Wrong registry credentials

An attacker with k8s admin privileges could supply unexpected credentials.
As stated above, this doesn't put image integrity at risk.
The only thing to be gained would be fine-granular metadata about pulls at the registry (that is, identifying individual client pods).

## Alternatives considered

### Credentials distributed with initdata

Credentials transferred via initdata suffer from the resource leak considerations described in [architecture/components/policies](../docs/docs/architecture/components/policies.md#evaluation).
This is unavoidable until we've a dedicated channel between guest components and Coordinator (see also [below](#credentials-distributed-by-coordinator)).
In the case of a public SaaS offering, for example, this leaks the cluster's registry credentials to the entire internet, which is undesirable.

### Deeper integration into Kubernetes

As outlined in the [Background](#background) section, there are other Kubernetes-native ways to get to registry credentials.
Implementing these for Contrast would be a large stretch, though:

* In order to retrieve secrets based on service accounts, the CLI would need direct access to the Kubernetes cluster.
* In order to use credential helpers, there would need to be a communication channel from the guest to the node.

Both seem disproportionately complex to implement, so we settle for the easier solution that still provides a path for users blocked by registry authentication.

### Global list of CA certificates

This is just too inflexible, given that most use cases will have a mix of internal and external registries and that manually adding web-PKI certificates is inconvenient.

### Credentials distributed by Coordinator

Instead of attaching credentials to the VM, we could make the Coordinator send them to the initializer, like the workload secret.
In the current architecture, this would result in a chicken-and-egg problem, because the initializer is a container that needs pulling.
While we could eventually move the initialization workflow into a guest component, this is a larger feature beyond the scope of this proposal.
Furthermore, this would introduce a dependency on remote systems during VM boot, which is hard to debug when it goes wrong.

#### Encrypted images

A topic that comes up frequently are encrypted images.
While at first glance it would seem appropriate to include encrypted images in this proposal, there are reasons why they're out of scope.

1. Confidentiality of the keys would require confidential conveyance, likely over an aTLS channel.
   At the very least, we'd need the initialization workflow changes outlined above.
2. In the `force_guest_pull` world, the host needs to have access to the layer keys (at least in general, there may be exceptions).

Since encryption in transit is already covered by CA certs and client credentials, encrypted images can be deferred until the necessary prerequisites are in place.

## Appendix

### Registry name matching

Kubernetes has a specification for how credentials are chosen for registries: <https://kubernetes.io/docs/concepts/containers/images/#config-json>.
This approach is unsuitable for our use case because it makes tightening of controls more difficult.
For example, we want to be able to pin certificates for a specific registry but not for others.

Instead, we implement an algorithm similar to [Go's handling of the `NO_PROXY` environment variable](https://pkg.go.dev/golang.org/x/net/http/httpproxy#Config).
The idea is to match a registry domain name against a list of suffixes and literal domains.
We ignore port numbers and IP addresses for now, since it's trivial to assign unique domain names to IP addresses in Kubernetes.
Support for those could be added later on without changing the semantics described below.

A literal domain is a fully-qualified domain name that starts with a non-empty label, for example `foo.bar.internal.`.
It matches a requested registry domain if it's exactly equal, ignoring the trailing dot.

A domain suffix is a fully-qualified domain name that starts with an empty label, for example `.foo.bar.internal.`.
It matches all subdomains, but not the domain `foo.bar.internal` itself.
A single `.`, the root domain, matches all valid domain names (since `.` itself isn't a valid domain name).

1. If there's a literal domain match, it always takes precedence over a suffix match.
2. If there is more than one suffix match, the longest match takes precedence.
   This ensures that we're not applying generic defaults to a registry with specific config.
3. If there is no match, we return the zero value for `Repository`, meaning defaults should be used.

The implementation shouldn't require the trailing dot, but add it implicitly if it's missing.
