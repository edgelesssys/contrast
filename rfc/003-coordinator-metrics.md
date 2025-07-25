# RFC 003: Contrast Metrics

The Coordinator should export metrics about itself to make it easier to observe
failures and detect internal problems. Metrics also allow generating usage
statistics and request durations of the UserAPI and the MeshAPI.

## The Problem

From a user perspective, there is currently no way of knowing what's going on
inside the Coordinator if the Coordinator has an internal failure without
looking into the Coordinator's logs. The only information that's available to
the user using the API is the received gRPC status code. If a workload startup
is failing, it's not immediately possible to tell the reason of the failure. In
the case of an attestation error, both the workload pod and the Coordinator have
to be examined in order to detect this.

## Solution

An easy way to automatically aggregate all this data would be to export metrics
in the form of [Prometheus](https://prometheus.io/docs/introduction/overview/)
or [OpenTelemetry](https://opentelemetry.io/docs/). A backend server could then
analyze/visualize the data exposed by the Coordinator.

### Solution 1: Prometheus

Prometheus collects [time series](https://en.wikipedia.org/wiki/Time_series)
data with key-value pairs and serves them as plain HTTP. When using Prometheus,
the Coordinator exposes gathered metrics on a specified port under the
`/metrics` endpoint. A Prometheus server can then scrape this endpoint.
Prometheus uses the pull model by default, meaning in a specified time interval,
it will scrape the metrics endpoint for new data. It's also possible to use a
[PushGateway](https://prometheus.io/docs/practices/pushing/), to push metrics
directly to an intermediary server which caches the data and a Prometheus server
can then scrape the PushGateway.

### Solution 2: OpenTelemetry

OpenTelemetry is a framework to create and manage
[traces, metrics, and logs](https://opentelemetry.io/docs/concepts/signals/). An
application can export data over _OTLP_ (OpenTelemetry Protocol), either
directly to a backend like Prometheus, or to an OpenTelemetry collector, which
can then export the data to any supporting backend. In addition to metrics,
which Prometheus also supports, OpenTelemetry can export traces, which include
start and end time stamps and nested traces, and logs, although this feature is
still experimental in the
[Golang SDK](https://opentelemetry.io/docs/languages/go/). All data is sent to
the specified collector endpoint by default.

---

When using OpenTelemetry, we would need to setup an OpenTelemetry collector
which exports the data to a backend like Prometheus or
[Jaeger](https://www.jaegertracing.io/docs/1.57/). Changing the backend would be
easy without changing the codebase in the Coordinator, as we only need to change
the exporter in the collector.

Prometheus is more limited in the gathered telemetry but probably sufficient for
our case. Prometheus data can directly be collected by the Prometheus server and
can always be integrated into OpenTelemetry as a receiver. OpenTelemetry can
then transform the received metrics from Prometheus and export them to any
backend.
[AKS already supports Prometheus monitoring](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/prometheus-metrics-overview)
as well. We will therefore use Prometheus for exposing metrics.

## Coordinator

### Exported metrics by Prometheus on the gRPC layer

When using the gRPC interceptor provided by the
[Golang gRPC Prometheus provider](https://pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus),
the following metrics are exported by the Coordinator by default:

- For the UserAPI:
  - Number of handled requests for methods SetManifest/GetManifest, including
    the error code.
  - Total number of messages received/sent.
  - Total number of started RPC connections.
- For the MeshAPI:
  - Number of handled requests for NewMeshCert, including the error code.
  - Total number of messages received/sent.
  - Total number of started RPC connections.
- Data about the scrape requests (number of requests / errors)

### Additional metrics

- Current manifest generation or if a manifest hasn't been set yet.
- Transport layer metrics for attestation failures including error messages.

## Service Mesh

The Service Mesh runs EnvoyProxy, which exposes metrics via an
[admin interface](https://www.envoyproxy.io/docs/envoy/latest/operations/admin).
To gather these metrics, the admin endpoint has to be activated, which apart
from exposing metrics can be used to query and modify different aspects of the
server. These metrics can also be queried in Prometheus format at the
`/stats/prometheus` endpoint.

## CI Monitoring

In order to monitor the CI cluster, we would need a Prometheus/OpenTelemetry
setup, which collects the metrics from E2E-tests. Because Prometheus uses the
pull model, a Prometheus server would send requests to all running workloads
which expose Prometheus metrics in a specified interval. It's possible that a
workload updates its metrics and is already deleted within this interval leading
to the Prometheus server losing some metrics. We don't expect this to be a
problem when monitoring the cluster over a longer period of time. Additionally,
we can wait for a few seconds before tearing down the E2E-setup to make sure the
metrics are scraped.

Apart from using exposed metrics to monitor the CI cluster, we could add simple
unit testing to already existing E2E-tests. Libraries for this already exist for
[Prometheus](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/testutil)
or
[OpenTelemetry](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest).
