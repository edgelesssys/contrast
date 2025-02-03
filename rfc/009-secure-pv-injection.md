# RFC 009: Secure PV Injection

## Objective

Setting up an encrypted volume mount using the `cryptsetup` container should be
automated during `contrast generate`. This can be done similar to injecting the
Initializer and Service Mesh components.

## Problem Statement

Contrast provides deployments with a shared workload secret which can be used to
setup an encrypted volume mount. This is useful for storing persistent sensitive
data (see [RFC 007](007-shared-workload-secret.md)). Currently, the process of
setting up an encrypted volume mount is by manually setting up an additional
init container running the `cryptsetup` subcommand of the Initializer as
described in the
[docs](https://docs.edgeless.systems/contrast/architecture/secrets). This can be
automated and made more user-friendly by automatically generating the necessary
YAML for the encrypted volume mount during `contrast generate`. This includes an
`EmptyDir` volume used for the shared state, and the necessary volume mounts in
the Initializer init container.

## Proposal

A new annotation `contrast.edgeless.systems/secure-pv` will be introduced which
specifies both the name of an existing volume device and the name of the
`EmptyDir` volume which will be used to provide the decrypted state of the PV:

```yaml
annotations:
  contrast.edgeless.systems/secure-pv: "volume-name:mount-name"
```

The `volume-name` is the name of an existing volume device which will be used by
the Initializer to setup an encrypted mount on a new `EmptyDir` volume specified
by `mount-name`. To use the encrypted volume, it needs to be mounted in the
workload containers manually. If no volume named `volume-name` is specified in
either the `volumes` or `volumeClaimTemplates` of the resource, `contrast
generate` will fail. Any existing volume with the name `mount-name` will be
replaced by an `EmptyDir` volume.

Currently, the Initializer uses the `cryptsetup` subcommand to setup an
encrypted volume. The container running the `cryptsetup` subcommand is running
separately from the Initializer container which attests to the Coordinator and
writes the Contrast secrets. Injecting the `cryptsetup` logic into the YAML, we
can simplify the process by:
- Running the `cryptsetup` subcommand in the same container as the Initializer
container.
- Deciding whether to quit the container after writing the secrets or to
continue with the `cryptsetup` and keep running.

In case the Initializer keeps running, we need a startup probe to check when the
`cryptsetup` is done. If the Initializer quits after writing the secrets, we can
skip the startup probe. The Initializer will decide whether to quit or to keep
running based on the environment variable `CRYPTSETUP_DEVICE` which will be set
if the corresponding annotation is set. By default, this argument is then set to
`/dev/csi0` and the Initializer will mount the PV to this device. The `EmptyDir`
volume specified by the `contrast.edgeless.systems/secure-pv` annotation will be
mounted to `/state` to complete the setup.

To summarize: if the `contrast.edgeless.systems/secure-pv` annotation is set,
the following changes will be made to the YAML:
- An `EmptyDir` volume will be added to the resource with the specified name
from the annotation.
- The Initializer `securityContext` will be updated to `privileged` to allow
mounting block devices.
- The Initializer container will mount the `EmptyDir` volume `mount-name` to
`/state` and the PV `volume-name` to `/dev/csi0`.
- The Initializer environment variables will be updated to include
`CRYPTSETUP_DEVICE=/dev/csi0`.
- The Initializer will have a startup probe to check if the `cryptsetup` is
done, new resource limits, as well as a restart policy of `Always`.

Skipping the `cryptsetup` injection can be done by skipping the Initializer
injection altogether. If the Initializer is skipped using the
`contrast.edgeless.systems/skip-initializer` annotation or the
`--skip-initializer` flag, the `contrast.edgeless.systems/secure-pv` annotation
will be ignored. Skipping the `cryptsetup` injection by simply not providing a
`contrast.edgeless.systems/secure-pv` annotation won't work, since the
Initializer will be replaced on `contrast generate` and the `CRYPTSETUP_DEVICE`
environment variable won't be set. In either case, if a user wants to manually
configure the encrypted PV, there are two options:
- The Initializer, the `EmptyDir` volume and corresponding volume mounts are
created manually, including the `CRYPTSETUP_DEVICE` environment variable for the
Initializer, startup probe, etc.
- Add an additional init container running the `cryptsetup` subcommand with the
same container image as the Initializer, as before.
