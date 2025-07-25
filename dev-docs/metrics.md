## Accessing metrics

For quick access to the metrics exposed by the Coordinator when debugging a
Contrast deployment, you can setup a Prometheus server with helm, using the
[Prometheus Community Helm Charts](https://github.com/prometheus-community/helm-charts):

```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
```

The default scrape interval is set to one minute. You may want to change this by
creating a `values.yml` file like this:

```yml
server:
  global:
    scrape_interval: 10s
```

You can then install the helm chart with the following command:

```sh
helm install <release-name> prometheus-community/prometheus -f values.yml
```
