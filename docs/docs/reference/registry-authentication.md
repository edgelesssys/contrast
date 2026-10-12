# Registry authentication

For the [Contrast image puller](../architecture/components/runtime.md#contrast-image-puller) to be able to pull images from a private registry, it must be configured with the required authentication details.
Contrast uses its own `TOML`-based configuration format for this purpose.
This configuration must be provided as a secret[^1] to the [node installer `DaemonSet`](../architecture/components/runtime.md#node-installer-daemonset).
This secret must be named `contrast-node-installer-imagepuller-config` and deployed into the same namespace as the node installer (`contrast-system` by default).
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

## Limitations

Currently, it's only possible to specify a single image puller auth configuration per runtime class, since only a single secret path can be configured for a runtime class.
Conversely, if this secret is present on a node, every node install will use it, granting deployments access to the configured registries.
This also extends to every newly installed runtime class.

Per default, all runtime classes use the same auth configuration, derived from the `contrast-node-installer-imagepuller-config` secret.
If you wish to use a different secret for a specific runtime class, you can change the `secretName` of the `DaemonSet` configuration in `runtime/runtime.yml` to a different name before applying the node installer, and put the image puller auth configuration into a secret with that name.

## Applying the configuration

The `contrast-node-installer-imagepuller-config` is configured as an optional `Secret` mount in the node installer.
This means that it must be applied before the runtime, applying it together with the runtime is often not enough.

If you need to modify or add the configuration after applying the runtime, you need to restart the node installer:

```sh
kubectl rollout restart daemonset/contrast-cc-metal-qemu-snp-deadbeef-nodeinstaller
kubectl rollout status -w daemonset/contrast-cc-metal-qemu-snp-deadbeef-nodeinstaller
```

## Configuration reference

The configuration is a TOML document at the secret key `contrast-imagepuller.toml`.
This document has global settings, as well as settings per registry.
The settings per registry are added under the registry name key, for example:

```toml
# Global settings go here.
[registries."ghcr.io."]
# Settings for ghcr.io go here.
```

:::note

The registry name must be a fully-qualified domain name, including the trailing dot, in lower case.

:::

### Registry matching and subdomains

The registry from the incoming image pull request is normalized to a fully-qualified domain and lower case.
This normalized domain name is then compared to the registry keys present in the configuration.
If there's a full match, that configuration is chosen for the pull.
If multiple matching registry configurations exist, for example if both `registries.".registry.corp."` and `registries.".corp."` have configuration values set, then the most specific match will be chosen.
In the example, this would be `registries.".registry.corp."`.
To specify a catch-all configuration, use the key `registries."."`.
If no registry matches, the image puller uses a default configuration (CA certificates for the web PKI and anonymous authentication).

#### IP addresses and ports

Currently, IP- and port-based configuration isn't supported.
Please use Kubernetes' built-in tools to assign unique domain names to IP addresses, and use these for image references instead.

### Configuration values

The following settings can be configured in a `contrast-imagepuller.toml` file.

#### `extra-env.HTTP_PROXY` {#extra-env-http-proxy}

Plain HTTP requests are redirected to this proxy unless exempted by [`NO_PROXY`](#extra-env-no-proxy).
The value is interpreted according to [Go's `httpproxy` configuration].

#### `extra-env.HTTPS_PROXY` {#extra-env-https-proxy}

HTTPS requests are redirected to this proxy unless exempted by [`NO_PROXY`](#extra-env-no-proxy).
The value is interpreted according to [Go's `httpproxy` configuration].

#### `extra-env.NO_PROXY` {#extra-env-no-proxy}

Requests to these servers bypass proxy settings.
This variable usually contains the proxies from [`HTTP_PROXY`](#extra-env-http-proxy) and [`HTTPS_PROXY`](#extra-env-https-proxy).
The value is interpreted according to [Go's `httpproxy` configuration].

[Go's `httpproxy` configuration]: https://pkg.go.dev/golang.org/x/net/http/httpproxy#Config

#### `registries.*.ca-certs` {#registry-ca-certs}

The acceptable CA certificates for registries matching this key, as a concatenation of PEM-encoded certificates.
This field serves two purposes, namely allowing connections to registries that aren't publicly trusted in the web PKI,
and to restrict who can ostensibly intercept and log traffic for metadata analysis.
If this field is empty, the web PKI CAs are used for the pull.
In order to disable pulls from a matching registry, set the field to a non-empty string that doesn't contain a PEM block.

#### `registries.*.client-cert` {#registry-client-cert}

A PEM-encoded client certificate that the image puller uses to authenticate itself towards the registry.
If a certificate is provided, the corresponding [private key](#registry-client-key) must be provided, too.

#### `registries.*.client-key` {#registry-client-key}

A PEM-encoded private key that the image puller uses to authenticate itself towards the registry.
If a private key is provided, the corresponding [certificate](#registry-client-cert) must be provided, too.

#### `registries.*.insecure-skip-verify` {#registry-insecure-skip-verify}

If set to true, this disables all TLS verification for matching registries.

:::warning

This setting is insecure and may lead to leaked credentials and image pull metadata.
Use only in trusted environments or for testing.
Image integrity is ensured by digest pinning and not affected by this setting.

:::

#### `registries.*.auth` {#registry-auth}

Base64-encoded HTTP basic auth credentials (`user:password`) for this registry.
This field is the same as the `auth` field of a registry in a [`containers-auth.json`] file, for example from `~/.docker/config.json`.

[`containers-auth.json`]: https://github.com/containers/image/blob/df7e80d2d19872b61f352a8a182ec934dc0c2346/docs/containers-auth.json.5.md

#### `registries.*.mirror` {#registry-mirror}

If your cluster uses an OCI registry mirror, you can configure the imagepuller to use that mirror, too.
The mirror address is a full HTTP URL, like `https://mirror.example.com:6443/path-prefix`.
If the `mirror` field is set, the other configuration options (credentials, CA certificates) apply to the connection to this mirror, not the original registry.
The mirror registry needs to implement the [relevant parts of the OCI spec](https://github.com/opencontainers/distribution-spec/blob/5e57cc0a07ea002e507a65d4757e823f133fcb52/spec.md?plain=1#L735-L753).
