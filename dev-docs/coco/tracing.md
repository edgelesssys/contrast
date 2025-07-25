# Tracing Kata

Kata has first-class support for tracing via OpenTelemetry. [^1] This document
describes how to collect the traces and display them in a web UI.

## Enabling tracing in the runtime

The Kata runtime can be configured for tracing by adding the following snippet
to the Kata configuration file (`configuration-qemu-snp.toml`, for example):

```toml
[runtime]
enable_tracing = true
```

The Kata runtime is then configured to send traces to an OpenTelemetry
collector, which needs to be set up separately.

## Running an OpenTelemetry Collector

Running an OpenTelemetry Collector is very simple when using the
[Jaeger all-in-one Docker container](https://www.jaegertracing.io/docs/1.6/getting-started/#all-in-one-docker-image).

This needs to be started on the runtime's host server:

```sh
docker run \
    -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
    -p 5775:5775/udp \
    -p 6831:6831/udp \
    -p 6832:6832/udp \
    -p 5778:5778 \
    -p 16686:16686 \
    -p 14268:14268 \
    -p 9411:9411 \
    jaegertracing/all-in-one:1.6.0
```

The web UI should then be available via http://localhost:16686.

## Viewing Traces

When reaching the web UI, one needs to select the "Kata" trace source on the
sidebar menu before clicking `Find traces`.

Then, a trace should appear for each sandbox / VM startup. This can then be
further inspected individually to get a grasp about the time individual
operations take.

[^1]: https://github.com/kata-containers/kata-containers/blob/main/docs/tracing.md
