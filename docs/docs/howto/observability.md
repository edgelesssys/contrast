# Observability

This section describes which metrics can be retrieved from the Contrast Coordinator and how to access them.

## Applicability

Monitoring metrics from the Contrast Coordinator helps you quickly identify issues in the gRPC layer or detect attestation failures.

## Prerequisites

1. [Set up cluster](./cluster-setup/aks.md)
2. [Install CLI](./install-cli.md)
3. [Deploy the Contrast runtime](./workload-deployment/runtime-deployment.md)

## How-to

The Contrast Coordinator can expose metrics in the
[Prometheus](https://prometheus.io/) format. Prometheus metrics
are numerical values associated with a name and additional key/values pairs,
called labels.

## Exposed metrics

The metrics can be accessed at the Coordinator pod on port 9102 using the `/metrics` endpoint.
By default, metrics are disabled. You can enable them by setting the `CONTRAST_METRICS`
environment variable.

The Coordinator exports gRPC metrics under the prefix `contrast_grpc_server_`.
These metrics are labeled with the gRPC service name and method name.
Metrics of interest include `contrast_grpc_server_handled_total`, which counts
the number of requests by return code, and
`contrast_grpc_server_handling_seconds_bucket`, which produces a histogram of\
request latency.

The gRPC service `userapi.UserAPI` records metrics for the methods
`SetManifest` and `GetManifest`, which get called when [setting the
manifest](./workload-deployment/set-manifest) and [verifying the
Coordinator](./workload-deployment/deployment-verification.md) respectively.

The `meshapi.MeshAPI` service records metrics for the method `NewMeshCert`, which
gets called by the [Initializer](../architecture/components/initializer.md) when starting a
new workload. Attestation failures from workloads to the Coordinator can be
tracked with the counter `contrast_meshapi_attestation_failures_total`.

The current manifest generation is exposed as a
[gauge](https://prometheus.io/docs/concepts/metric_types/#gauge) with the metric
name `contrast_coordinator_manifest_generation`. If no manifest is set at the
Coordinator, this counter will be zero.

## Service mesh metrics

The [Service Mesh](../architecture/components/service-mesh.md) can be configured to expose
metrics via its [Envoy admin
interface](https://www.envoyproxy.io/docs/envoy/latest/operations/admin). Be
aware that the admin interface can expose private information and allows
destructive operations to be performed. To enable the admin interface for the
Service Mesh, set the annotation
`contrast.edgeless.systems/servicemesh-admin-interface-port` in the configuration
of your workload. If this annotation is set, the admin interface will be started
on this port.

To access the admin interface, the ingress settings of the Service Mesh have to
be configured to allow access to the specified port (see [Configuring the
Proxy](../architecture/components/service-mesh.md#configuring-the-proxy). All metrics will be
exposed under the `/stats` endpoint. Metrics in Prometheus format can be scraped
from the `/stats/prometheus` endpoint.
