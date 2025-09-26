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

**NOTE**: This design depends on the [initdata processor](https://github.com/kata-containers/kata-containers/issues/11532), which isn't merged upstream at the time of writing.

### Changes to the `imagepuller`

We add support for an optional configuration file to the `imagepuller`, with the following structure:

```go
import "github.com/google/go-containerregistry/pkg/authn"

type Config struct {
    Auths map[string]authn.AuthConfig // key is the OCI registry
    CA [][]byte // list of trusted PEM-encoded CA certificates
    ExtraEnv map[string]string
}
```

On startup, the `imagepuller` reads the config file from `/run/measured-cfg/imagepuller.TBD`, where it was provisioned by the initdata processor.
It takes the `Auths` and stores them in a field, and creates a [`tls.Config`].
If `CA` isn't empty, it overrides the `RootCAs` field of the [`tls.Config`].
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

### Changes to the CLI

During `contrast generate`, the CLI walks over the Kubernetes resources looking for a secret with the appropriate `type` field.
It combines the individual `dockerconfigjson` messages into the `Auths` field of `imagepuller.Config`.
Two new repeatable command line flags, `--image-pull-cert` and `--image-pull-env`, populate the remaining fields of the config file.
The config is then serialized and attached to the initdata produced by `genpolicy`.

### Security considerations

#### Credentials visible in initdata annotation

Credentials transferred via initdata suffer from the resource leak considerations described in [architecture/components/policies](../docs/docs/architecture/components/policies.md#evaluation).
This is unavoidable until we've a dedicated channel between guest components and Coordinator (see also [below](#credentials-distributed-by-coordinator)).

#### CA certificates

The CA certificates aren't required for image integrity - that's accomplished by pinning references.
Rather, the CA certificate option serves two purposes:

1. Allow connecting to a registry that isn't publicly trusted in the web PKI.
2. Restrict who can ostensibly intercept and log traffic for metadata analysis (for example, what images are loaded).

## Alternatives considered

### Configuration file

The information in an `imagepuller` config file is usually global for a deployment site, and would thus naturally fit into a config file for the CLI.
However, the question whether we want a config file for the CLI and how we should go about adding one is beyond the scope of this proposal.
As soon as we've a config file, we can allow registry authentication there while keeping or deprecating the mechanism proposed here.

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
Given that the CA cert is defense in depth and doesn't affect the CC-security (see [CA certificates](#ca-certificates)), it's probably fine to defer such a feature until we've configuration files.

### Credentials distributed by Coordinator

Instead of sending the credentials with initdata, we could attach them to the manifest and hand them to the initializer, like the workload secret.
In the current architecture, this would result in a chicken-and-egg problem, because the initializer is a container that needs pulling.
While we could eventually move the initialization workflow into a guest component, this is a larger feature beyond the scope of this proposal.
But whenever we reach the point where the initializer runs as a guest component, we can change the delivery mechanism for the image puller config without user-visible changes.

#### Encrypted images

A topic that comes up frequently are encrypted images.
While at first glance it would seem appropriate to include encrypted images in this proposal, there are reasons why they're out of scope.

1. Confidentiality of the keys would require confidential conveyance, likely over an aTLS channel.
   At the very least, we'd need the initialization workflow changes outlined above.
2. In the `force_guest_pull` world, the host needs to have access to the layer keys (at least in general, there may be exceptions).

Since encryption in transit is already covered by CA certs and client credentials, encrypted images can be deferred until the necessary prerequisites are in place.
