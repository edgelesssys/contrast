# RFC 011: Registry Authentication

## Background

### Registry credentials

Most container registries used in enterprise environments require authentication to pull images.
There are several ways in which these credentials can be supplied to the container runtime:

1. Image pull secret referenced directly in the pod spec: <https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/>.
2. Image pull secret attached to the service account: <https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account>. This indirectly populates the `imagePullSecrets` field of all pod specs using this service account.
3. Global credentials in the container runtime configuration: <https://github.com/containerd/containerd/blob/5bcf77a/docs/cri/registry.md?plain=1#L37-L40>.
4. TODO(burgerdev): also mention https://kubernetes.io/docs/tasks/administer-cluster/kubelet-credential-provider/

Image pull secrets come in [two formats](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets).
The `dockercfg` format has been [removed](https://github.com/docker/cli/pull/2504) by Docker itself, but this was not too long ago and it's likely still widely used.

### Registry certificates

Internal registries often use certificates that don't chain back to a web PKI root CA, but an internal one.
Since Kubernetes does not have an API for specifying trusted registry CAs, such CA certificates need to be [configured at the container runtime](https://github.com/containerd/containerd/blob/5bcf77a55038ad658c57fdecc48af54935a0d42f/docs/cri/config.md?plain=1#L744).

## Requirements

1. Users must be able to pull images from registries that require authentication.
2. Users must be able to pull images from registries that don't participate in web PKI.
3. Users must be able to pull images through HTTP proxies.

## Design

**NOTE**: This design depends on the [initdata processor](https://github.com/kata-containers/kata-containers/issues/11532), which is not merged upstream at the time of writing.

### Changes to the `imagepuller`

We add support for a configuration file to the `imagepuller`, with the following structure:

```go
import "github.com/google/go-containerregistry/pkg/authn"

type Config struct {
    Auths map[string]authn.AuthConfig // key is the OCI registry
    CA [][]byte // list of trusted PEM-encoded CA certificates
    ExtraEnv map[string]string
}
```

On startup, the `imagepuller` reads the config file from `/run/measured-cfg/imagepuller.TBD`.
It takes the `Auths` and stores them in a field, and creates a `tls.Config`.
If `CA` is not empty, it overrides the `RootCAs` field of the `tls.Config`.
Each key-value pair in `ExtraEnv` is added to the `imagepuller`s own environment.

When a `PullImage` request comes in, the `imagepuller` extracts the registry part from the image reference and looks it up in `Auths`.
If present, it wraps the [`authn.AuthConfig`] with [`authn.FromConfig`] and passes it as an option [`remote.WithAuth`].
The `tls.Config` and `http.ProxyFromEnvironment` are used to construct an `http.Transport` that's passed as an option [`remote.WithTransport`].

[`authn.AuthConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#AuthConfig
[`authn.FromConfig`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#FromConfig
[`remote.WithAuth`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithAuth
[`remote.WithTransport`]: https://pkg.go.dev/github.com/google/go-containerregistry/pkg/v1/remote#WithTransport

### Changes to the CLI

### Security considerations

#### Credentials visible in initdata annotation

## Alternatives considered

### CA per registry

### Credentials distributed by Coordinator
