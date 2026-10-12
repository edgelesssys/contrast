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

#### Creating an image puller configuration

Various configuration options are available, both globally and per-registry.
These options are documented in the [image puller configuration reference](../reference/registry-authentication.md).

The following script generates a valid configuration file and makes it available to the node installer.

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

kubectl create secret generic -n contrast-system --from-file=contrast-imagepuller.toml=contrast-imagepuller.toml contrast-node-installer-imagepuller-config
```

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
Registry domains other than `very-secure.registri.es` are strictly forbidden.

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
ca-certs = "no PEM here means no CA certificates, so other registries always fail the TLS handshake"

[registries.".very-secure.registri.es."]
ca-certs = "this entry disables all subdomains of very-secure.registri.es."
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

In this scenario, we're using the [mirror registry provided by k3s](https://docs.k3s.io/installation/registry-mirror).
All images running in Contrast should be pulled through that mirror.
For convenience, we're using the name of a single node to refer to that registry and switch off TLS certificate validation.

```toml
[registries."."]
mirror = "https://my-k3s-node-1:6443"
insecure-skip-verify = true
```

##### Example 5

If no image puller configuration is provided or if it's empty, the behavior for all registries is to use no authentication, and to use and trust web PKI.
