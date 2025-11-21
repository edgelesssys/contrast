# Overview of Contrast elements in Kubernetes YAML

This document provides an overview of the Contrast elements that can be used in Kubernetes YAML files, particularly the Contrast specific annotations that can be applied to workloads.

## Overview of Contrast annotations

Before running the `contrast generate` command, you can customize its behavior by using various annotations, like skipping the Initializer injection or settings up the Service Mesh.
These annotations can be added to the workload's Pod (Pod Template) metadata.
The following table gives an overview of the available annotations and their purpose.

| Annotation                                                   | Description                                                                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `contrast.edgeless.systems/skip-initializer`                 | [Skip the Initializer injection for this workload.](../howto/workload-deployment/generate-annotations.md#skip-initializer-annotation) |
| `contrast.edgeless.systems/servicemesh-ingress`              | [Setup the Service Mesh ingress for this workload.](components/service-mesh.md#configuring-the-proxy)                                 |
| `contrast.edgeless.systems/servicemesh-egress`               | [Setup the Service Mesh egress for this workload.](components/service-mesh.md#configuring-the-proxy)                                  |
| `contrast.edgeless.systems/servicemesh-admin-interface-port` | [Enable the Envoy admin interface for the Service Mesh on the specified port.](components/service-mesh.md#configuring-the-proxy)      |
| `contrast.edgeless.systems/secure-pv`                        | [Enable secure storage for the workload by setting up a LUKS-encrypted volume.](secrets.md#secure-persistence)                        |
| `contrast.edgeless.systems/workload-secret-id`               | [Specify the `workloadSecretID` to use for this workload.](secrets.md#workload-secrets)                                               |
| `contrast.edgeless.systems/image-store-size`                 | [Specify the size of the secure image store. Set to `0` to disable.](../howto/secure-image-store.md)                                  |
