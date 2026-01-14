# Registry authentication

This guide shows how to set up registry credentials for Contrast.

## Applicability

This guide is relevant if you need to authenticate with a container registry for pulling images.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)
2. [Install CLI](./install-cli.md)
3. [Deploy the Contrast runtime](./workload-deployment/runtime-deployment.md)
4. [Prepare deployment files](./workload-deployment/deployment-file-preparation.md)

## How-to

### Contrast CLI

The Contrast CLI, specifically the `contrast generate` subcommand, needs access to the registry to derive policies for the referenced container images.
The CLI authenticates to the registry using the [`docker_credential`](https://crates.io/crates/docker_credential) crate.
This crate searches some default locations for a registry authentication file, so it should find credentials created by `docker login` or `podman login`.

The only authentication method that's currently supported is `Basic` HTTP authentication with user name and password (or personal access token).
Identity token flows, such as the default mechanism of plain `docker login`, don't work.
Basic authentication can be forced with `docker login -u $REGISTRYUSER`.

You can override the credentials file used by Contrast by setting the environment variable `DOCKER_CONFIG`. This is useful for creating a credential file from scratch, as shown in the following script:

```sh
#!/bin/sh

registry="<put registry here>"
user="<put user id here>"
password="<put client secret here>"

export DOCKER_CONFIG=$(mktemp)

cat >"${DOCKER_CONFIG}" <<EOF
{
        "auths": {
                "$registry": {
                        "auth": "$(printf "%s:%s" "$user" "$password" | base64 -w0)"
                }
        }
}
EOF

contrast generate "$@"
```

### Confidential guest VMs

For the [Contrast image puller](../architecture/components/runtime.md#contrast-image-puller) to be able to pull images from a private registry, it must be configured with the required authentication details.
Contrast uses its own `TOML`-based configuration format for this purpose.
This configuration must be provided as a secret[^1] to the [node installer `DaemonSet`](../architecture/components/runtime.md#node-installer-daemonset).
This secret must be named `contrast-node-installer-imagepuller-config` and belong to the `kube-system` namespace for the node installer to recognize and use it.
The node installer then handles installing the secret in the runtime directory on the host, from where the runtime then forwards it to the confidential pod VM, allowing the image puller to read it.

[^1]: Kubernetes secrets are accessible to anyone with API access to the node, as well as anyone with access to `etcd` ([source](https://kubernetes.io/docs/concepts/configuration/secret/)).
    Anyone with these permissions is additionally able to change the configuration's content, with no way for Contrast to verify the integrity of the configuration eventually received by the image puller.

    This allows an attacker to redirect image pull requests and configure an attacker-controlled registry to appear as legitimate.
    The attacker could then, for example, log which images are being pulled.

    This does **not** allow an attacker to serve malicious images or alter the contents of valid images, since irrespective of the image puller's configuration,
    pulled images are integrity-protected through their mandatory pinned digest.

:::warning

Since the image puller configuration is annotated to the pod, it can be retrieved by any role with `get` or `list` permission for pods.
This may result in an unexpected leak of sensitive information.

:::

#### Limitations

Currently, it's only possible to specify a single image puller auth configuration per runtime class, since only a single secret path can be configured for a runtime class.
Conversely, if this secret is present on a node, every node install will use it, granting deployments access to the configured registries.
This also extends to every newly installed runtime class.

Per default, all runtime classes use the same auth configuration, derived from the `contrast-node-installer-imagepuller-config` secret.
If you wish to use a different secret for a specific runtime class, you can change the `secretName` of the `DaemonSet` configuration in `runtime/runtime.yml` to a different name before applying the node installer, and put the image puller auth configuration into a secret with that name.

#### Creating an image puller configuration

Various configuration options are available, both globally and per-registry.
The following table shows the available global configuration options.

| Option                                                   | Description                                                                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `extra-env.HTTP_PROXY` | proxy to use for plain HTTP requests |
| `extra-env.HTTPS_PROXY` | proxy to use for HTTPS requests |
| `extra-env.NO_PROXY` | registry domains for which the proxy should be bypassed |

Please see [the Go `httpproxy` documentation](https://pkg.go.dev/golang.org/x/net/http/httpproxy#Config) for details on the usage and semantics of these environment variables.

For each individual registry `registry.corp`, the following options are available under `registries."registry.corp."`:

| Option                                                   | Description                                                                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `ca-certs` | a newline-concatenated list of PEM-encoded certificates |
| `insecure-skip-verify` | disable transport security |
| `auth` | base64-encoded HTTP basic auth credentials authenticating the user with the registry |

The `auth` credentials use the same format as shown above for the Contrast CLI.
The following script generates a valid configuration file.

```sh
#!/bin/sh

registry="<put registry here>"
user="<put user id here>"
password="<put client secret here>"

cat > "contrast-imagepuller.toml" <<EOF
[registries]
[registries."$registry."]
auth = "$(printf "%s:%s" "$user" "$password" | base64 -w0)"
EOF
```

The ability to specify CA certificates mainly serves two purposes, namely allowing connections to registries that aren't publicly trusted in the web PKI,
and to restrict who can ostensibly intercept and log traffic for metadata analysis.

A number of example configurations for various use-case scenarios are shown below.

##### Example 1

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

##### Example 2

In this scenario, all container images are served from a public registry.
However, the image owner wants to make sure that the traffic can't be intercepted by rogue CAs.
Other registries are strictly forbidden.

```toml
[registries."very-secure.registri.es."]
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

##### Example 3

In this scenario, there's an HTTP-only registry deployed into the cluster for ease of use.
Transport security for this internal registry isn't important to the operators.
Other registries should be used anonymously, but with TLS.

```toml
[registries."registry.default.svc.cluster.local."]
insecure-skip-verify = true
```

##### Example 4

If no image puller configuration is provided or if it's empty, the behavior for all registries is to use no authentication, and to use and trust web PKI.

#### Registry matching and subdomains

Registry domains are specified as fully qualified domain names.
Note the trailing dot in the examples above.
For a registry-specific configuration to be applied to a pull request, the image's registry must end exactly in the configuration's name.
A configuration above for `registries.".registry.corp."` will be applied to any and all registries available on subdomains of `registry.corp`, but not to `registry.corp` itself.
Pulling the image `example.registry.corp/example/image@sha256:...` will use the configuration given under `registries.".registry.corp."`, but `registry.corp/example/image@sha256:...` won't.

Additionally, `example.registry.corp` must be able to prove its identity by successfully completing a TLS handshake using one of the explicitly configured certificates.
If no certificates are configured, the hosts default web PKI certificates are used.

#### Multiple matches and global configuration

If multiple matching registry configurations exist, for example if both `registries.".registry.corp."` and `registries.".corp."` have configuration values set, then the most specific match will be chosen.
In the example, this would be `registries.".registry.corp."`.

To specify a catch-all configuration, use the key `registries."."`.
The option `registries.".".ca-certs` can be used to disable authentication with all unknown registries:

```toml
[registries."."]
ca-certs = '''
This option set, but no certificates provided, means no TLS handshake will succeed.
'''
```

#### IP addresses and ports

Currently, IP- and port-based configuration isn't supported.
Please use Kubernetes' built-in tools to assign unique domain names to IP addresses, and use these for image references instead.
