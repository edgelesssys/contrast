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
- `files[*].path`: Target location of the file on the host filesystem.
- `files[*].integrity`: Expected Subresource Integrity (SRI) digest of the file. Only required if URL starts with `http://` or `https://`.
- `runtimeHandlerName`: Name of the container runtime.

Consider the following example:

```json
{
    "files": [
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers.img",
            "path": "/opt/edgeless/share/kata-containers.img",
            "integrity": "sha256-EdFywKAU+xD0BXmmfbjV4cB6Gqbq9R9AnMWoZFCM3A0="
        },
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers-igvm.img",
            "path": "/opt/edgeless/share/kata-containers-igvm.img",
            "integrity": "sha256-E9Ttx6f9QYwKlQonO/fl1bF2MNBoU4XG3/HHvt9Zv30="
        },
        {
            "url": "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/cloud-hypervisor-cvm",
            "path": "/opt/edgeless/bin/cloud-hypervisor-snp",
            "integrity": "sha256-coTHzd5/QLjlPQfrp9d2TJTIXKNuANTN7aNmpa8PRXo="
        },
        {
            "url": "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2",
            "path": "/opt/edgeless/bin/containerd-shim-contrast-cc-v2",
        }
    ],
    "runtimeHandlerName": "contrast-cc"
}
```
