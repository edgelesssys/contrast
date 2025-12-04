# CSI resources

This directory contains resources for deploying the [csi-driver-host-path].

[csi-driver-host-path]: https://github.com/kubernetes-csi/csi-driver-host-path

## Usage

The `kustomization.yaml` is ready to be applied to our CI runners.
It will install the CSI driver into the `csi-system` namespace and set up the necessary RBAC resources.

```bash
kubectl apply -k .
```

## Provenance

The official deployment flow is a [shell script] and there are no stable resource definitions.
Therefore, we observed the rendered resources once, at [this commit], and are tracking them in this directory instead.

If necessary, this can be reproduced by logging the directory content of `TEMP_DIR` whenever `kubectl apply` is executed.
However, we don't expect too many changes here, since the driver isn't released in lockstep with Kubernetes and releases are infrequent.

[shell script]: https://github.com/kubernetes-csi/csi-driver-host-path/blob/69142b68de86efed13e615c2ed9c98b62b5234a2/deploy/util/deploy-hostpath.sh
[this commit]: https://github.com/kubernetes-csi/csi-driver-host-path/commit/69142b68de86efed13e615c2ed9c98b62b5234a2
