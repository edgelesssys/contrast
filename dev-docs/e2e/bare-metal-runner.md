## Bare-metal runner specification
To run our e2e test in with the real bare-metal runner specification a ConfigMap named `bm-tcb-specs` is added to both e2 clusters: `m50-ganondorf` and `discovery`.
Having the ConfigMap prevents using committed values in the e2e tests directly, which could otherwise lead to backporting problems.

The `bm-tcb-specs` ConfigMap wraps the [`tcb-specs.json`](/dev-docs/e2e/tcb-specs.json), sharing TDX and SNP bare-metal specifications.
While the ConfigMap stores both runner specifications the [patchReferenceValues()](https://github.com/edgelesssys/contrast/blob/main/e2e/internal/contrasttest/contrasttest.go#L254-L283) function will only use the platform-specific reference values for overwriting.

### Setting up e2e clusters / Updating `tcb-specs.json`
We expect the e2e clusters not to be destroyed frequently, thus the ConfigMap is stored persistently for the e2e test. In case of setting up the e2e clusters again or an update to the bare-metal runner specifications is required, the ConfigMap has to be applied with:

``` bash
kubectl create configmap bm-tcb-specs --from-file=tcb-specs.json -n default
```
