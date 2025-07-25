# Registry authentication

This guide shows how to set up registry credentials for Contrast.

## Contrast CLI

The Contrast CLI, specifically the `contrast generate` subcommand, needs access
to the registry to derive policies for the referenced container images. The CLI
authenticates to the registry using the
[`docker_credential`](https://crates.io/crates/docker_credential) crate. This
crate searches some default locations for a registry authentication file, so it
should find credentials created by `docker login` or `podman login`.

The only authentication method that's currently supported is `Basic` HTTP
authentication with user name and password (or personal access token). Identity
token flows, such as the default mechanism of plain `docker login`, don't work.
Basic authentication can be forced with `docker login -u $REGISTRYUSER`.

You can override the credentials file used by Contrast by setting the
environment variable `DOCKER_CONFIG`. This is useful for creating a credential
file from scratch, as shown in the following script:

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

## AKS

On AKS, images are pulled on the worker nodes using credentials available to
Kubernetes and containerd. Follow the
[official instructions](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
to set up registry authentication with image pull secrets.

## Bare metal

On bare metal, images are pulled within the confidential guest, which doesn't
receive credentials from the host yet. You can work around this by mirroring the
required images to a private registry that's only exposed to the cluster. Such a
registry needs to have a valid TLS certificate that's trusted in the web PKI
(issued by Let's Encrypt, for example).
