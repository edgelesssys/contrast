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

The overall idea of this design is to supply an image puller configuration to the VM that is _not measured_.
Reason for this is that the image pull configuration is a workload owner secret that should not be visible to (unauthenticated) verifiers - see [Alternatives considered](#alternatives-considered).

### Changes to the `imagepuller`

We add support for an optional configuration file to the `imagepuller`, with the following structure:

```go
import "github.com/google/go-containerregistry/pkg/authn"

type InsecureConfig struct {
    Auths map[string]authn.AuthConfig // key is the OCI registry
    CA [][]byte // list of trusted PEM-encoded CA certificates
    InsecureSkipVerify bool // applies to all registry connections
    ExtraEnv map[string]string
}
```

On startup, the `imagepuller` reads the config file from `/run/insecure-cfg/imagepuller.TBD`.
It takes the `Auths` and stores them in a field, and creates a [`tls.Config`].
If `CA` isn't empty, it overrides the `RootCAs` field of the [`tls.Config`].
If `InsecureSkipVerify` is true, the field is also set on the [`tls.Config`].
Each key-value pair in `ExtraEnv` is added to the `imagepuller`s own environment.

When a `PullImage` request comes in, the `imagepuller` extracts the registry part from the image reference and looks it up in `Auths`.
If present, it wraps the [`authn.AuthConfig`] with [`authn.FromConfig`] and passes it as an option [`remote.WithAuth`].
The [`tls.Config`] and [`http.ProxyFromEnvironment`] are used to construct an [`http.Transport`] that's passed as an option [`remote.WithTransport`].

[`tls.Config`]: https://pkg.go.dev/crypto/tls#Config
[`http.ProxyFromEnvironment`]: https://pkg.go.dev/net/http#ProxyFromEnvironment
[`http.Transport`]: https://pkg.go.dev/net/http#Transport
[`authn.AuthConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#AuthConfig
[`authn.FromConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#FromConfig
[`remote.WithAuth`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithAuth
[`remote.WithTransport`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithTransport

### Changes to the node-installer

The node-installer gets another optional volume mount, similar to the [target config](https://github.com/edgelesssys/contrast/blob/6ef858031759966dfd3cdeda2b4570bed45fdcda/internal/kuberesource/parts.go#L126-L131) but using a secret.
The secret contains one key, `imagepuller.TBD`, containing a serialized version of `InsecureConfig` defined above.
This secret is created by the k8s administrator (or the workload owner) before applying the runtime.
If the node-installer finds a mounted secret, it writes the content into `/opt/edgeless/contrast-cc-*/etc/host-config/imagepuller.TBD`.

node-installer operations are intended to be idempotent.
In order to change the imagepuller configuration, the k8s administrator only needs to change the secret and restart the `DaemonSet`.

### Changes to the Kata runtime

During sandbox creation, the Kata runtime packs all files under `/opt/edgeless/contrast-cc-*/etc/host-config` into a device and attaches it to the VM.
This code should follow the initdata device provisioning logic very closely, but use a different magic identifier (TBD).

### Changes to the image

We add a new functionality to the initdata-processor:
After verifying initdata, the initdata-processor scans for the device introduced above.
The content of this device is copied over to `/run/insecure-cfg`, without any integrity checks.
The name is generic to allow for future use cases outside of image pulling.

### Security considerations

We need to be careful about the trust we put into configuration by the host.
For example, it would be a bad idea to allow switching off digest validation depending on such host configuration.
However, the fields in the imagepuller config are not a risk to guest integrity, which is explained in the following subsections.

#### Wrong CA certificates

The CA certificates aren't required for image integrity - that's accomplished by pinning references.
Rather, the CA certificate option serves two purposes:

1. Allow connecting to a registry that isn't publicly trusted in the web PKI.
2. Restrict who can ostensibly intercept and log traffic for metadata analysis (for example, what images are loaded).

An attacker with k8s admin privileges could configure CA certificates to serve registry requests from unexpected endpoints.
They'd still need to serve the correct, pinned image, so they can only record traffic passively.
However, the image data is already exposed to the host (due to containerd's host pull), and so are the registry credentials.

The argument is very similar for `InsecureSkipVerify` and proxy env vars.

#### Wrong registry credentials

An attacker with k8s admin privileges could supply unexpected credentials.
As stated above, this does not put image integrity at risk.
The only thing to be gained would be fine-granular metadata about pulls at the registry (i.e., identifying individual client pods).

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

### Extend list of CAs

Instead of overriding `RootCAs`, we could instead extend the list of system roots.
However, users have good reasons not to trust the system roots, so we would need to add another knob to allow that.
It's probably easier - to implement and to audit - to let users specify the full list of trusted roots.

### CA per registry

Instead of having a list of globally trusted CAs, we could have CAs per registry.
This would allow fine-grained control over who can authenticate which image source.
On the other hand, this introduces complexity for the user, because they need to express this relationship somehow.
Given that the CA cert is defense in depth and doesn't affect the CC-security (see [CA certificates](#wrong-ca-certificates)), it's probably fine to defer such a feature until we've configuration files.

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
