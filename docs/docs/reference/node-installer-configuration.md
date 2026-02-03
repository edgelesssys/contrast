# Node installer configuration

The Contrast node installer uses an optional `ConfigMap` to customize the installation process.
Such a `CofigMap` must be called `contrast-node-installer-target-config` and reside in the same namespace as the node installer.
The `ConfigMap` is expected to contain one or more of the entries listed in the following sections.
If an entry is absent, the documented default value will be used.
Note that an absent entry may be different from an entry with an empty value.

:::warning

The `ConfigMap` should only contain keys listed below.
Other keys are reserved for future extensions.

:::

## Applying the configuration

The `contrast-node-installer-target-config` is configured as an optional `ConfigMap` mount in the node installer.
This means that it must be applied before the runtime, applying it together with the runtime is often not enough.

If you need to modify or add the configuration after applying the runtime, you need to restart the node installer:

```sh
kubectl rollout restart daemonset/contrast-cc-metal-qemu-snp-deadbeef
kubectl rollout status -w daemonset/contrast-cc-metal-qemu-snp-deadbeef
```

## Configuration values

### `containerd-config-path`

The path to the containerd config relative to the node's file system root.
This configuration will be amended with an entry for the Contrast runtime class handler.
If the path ends with the `.tmpl` extension, the node installer assumes a [k3s configuration template](https://docs.k3s.io/advanced#configuring-containerd), but overwrites it with the current rendered version.

The default path is `etc/containerd/config.toml`.

### `systemd-unit-name`

A comma-separated list of systemd units to restart.
The node installer walks through this list until it finds a unit that's present on the node, and then attempts to restart it.
If none of the units exist, the node installer will fail the installation, exit and retry again in the next pod instance.

If the list is empty, systemd units won't be restarted by the Contrast runtime.
In this case, an administrator must ensure that containerd picks up the new configuration file.

By default, the node installer attempts to restart `containerd.service`.
