# Observability

The Contrast Coordinator can expose metrics in the
[Prometheus](https://prometheus.io/) format. These can be monitored to quickly
identify problems in the gRPC layer or attestation errors. Prometheus metrics
are numerical values associated with a name and additional key/values pairs,
called labels.

## Exposed metrics

The metrics can be accessed at the Coordinator pod at the port specified in the
`CONTRAST_METRICS_PORT` environment variable under the `/metrics` endpoint. By
default, this environment variable isn't specified, hence no metrics will be
exposed.

The Coordinator starts two gRPC servers, one for the user API on port `1313` and
one for the mesh API on port `7777`. Metrics for both servers can be accessed
using different prefixes.

All metric names for the user API are prefixed with `contrast_userapi_grpc_server_`.
Exposed metrics include the number of  handled requests of the methods
`SetManifest` and `GetManifest`, which get called when [setting the
manifest](../deployment#set-the-manifest) and [verifying the
Coordinator](../deployment#verify-the-coordinator) respectively. For each method
you can see the gRPC status code indicating whether the request succeeded or
not and the request latency.

For the mesh API, the metric names are prefixed with `contrast_meshapi_grpc_server_`. The
metrics include similar data to the user API for the method `NewMeshCert` which
gets called by the [Initializer](../components#the-initializer) when starting a
new workload. Attestation failures from workloads to the Coordinator can be
tracked with the counter `contrast_meshapi_attestation_failures`.

The current manifest generation is exposed as a
[gauge](https://prometheus.io/docs/concepts/metric_types/#gauge) with the metric
name `contrast_coordinator_manifest_generation`. If no manifest is set at the
Coordinator, this counter will be zero.

## Service Mesh metrics

The [Service Mesh](../components/service-mesh.md) can be configured to expose
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
Proxy](../components/service-mesh#configuring-the-proxy)). All metrics will be
exposed under the `/stats` endpoint. Metrics in Prometheus format can be scraped
from the `/stats/prometheus` endpoint.
