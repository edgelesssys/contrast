# Contrast node installer

This program runs as a `daemonset` on every CC-enabled node of a Kubernetes cluster.
It expects the host filesystem of the node to be mounted under `/host`.
On start, it will read a configuration file under `$CONFIG_DIR/contrast-node-install.json` and install binary artifacts on the host filesystem according to the configuration.
After installing binary artifacts, it installs and patches configuration files for the Contrast runtime class `contrast-cc-isolation` and restarts containerd.

## Configuration

By default, the installer ships with a config file under `/config/contrast-node-install.json`, which takes binary artifacts from the container image.
If desired, you can replace the configuration using a Kubernetes configmap by mounting it into the container.

- `files`: List of files to be installed.
- `files[*].url`: Source of the file's content. Use `http://` or `https://` to download it or `file://` to copy a file from the container image.
- `files[*].path`: Target location of the file on the host filesystem. The `@@runtimeBase@@` placeholder can be used to get a unique per-runtime-handler path.
    For example, `@@runtimeBase@@/foo` will resolve to `/opt/edgeless/contrast-cc-<platform>-<runtime-hash>/foo`, where `<platform>` is the platform the node-installer is deployed on,
    and `<runtime-hash>` is a hash of all relevant runtime components, so that it's unique per-version.
- `files[*].integrity`: Expected Subresource Integrity (SRI) digest of the file. Only required if URL starts with `http://` or `https://`.
- `debugRuntime`: If set to true, enables [serial console access via `vsock`](../dev-docs/serial-console.md). A special, debug-capable IGVM file has to be used for this to work.

Consider the following example:

```json
{
    "files": [
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers.img",
            "path": "@@runtimeBase@@/kata-containers.img",
            "integrity": "sha256-EdFywKAU+xD0BXmmfbjV4cB6Gqbq9R9AnMWoZFCM3A0="
        },
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers-igvm.img",
            "path": "@@runtimeBase@@/kata-containers-igvm.img",
            "integrity": "sha256-E9Ttx6f9QYwKlQonO/fl1bF2MNBoU4XG3/HHvt9Zv30="
        },
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/cloud-hypervisor-cvm",
            "path": "@@runtimeBase@@/cloud-hypervisor-snp",
            "integrity": "sha256-coTHzd5/QLjlPQfrp9d2TJTIXKNuANTN7aNmpa8PRXo="
        },
        {
            "url": "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2",
            "path": "@@runtimeBase@@/containerd-shim-contrast-cc-v2",
        }
    ],
    "debugRuntime": false
}
```
