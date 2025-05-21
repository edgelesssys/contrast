# Product features

Contrast simplifies the deployment and management of Confidential Containers, offering optimal data security for your workloads while integrating seamlessly with your existing Kubernetes environment.

From a security perspective, Contrast employs the [Confidential Containers](confidential-containers.md) concept and provides [security benefits](security-benefits.md) that go beyond individual containers, shielding your entire deployment from the underlying infrastructure.

From an operational perspective, Contrast provides the following key features:

* **Managed Kubernetes compatibility**: Initially compatible with Azure Kubernetes Service (AKS), Contrast is designed to support additional platforms such as AWS EKS and Google Cloud GKE as they begin to accommodate confidential containers.

* **Lightweight installation**: Contrast can be integrated as a [day-2 operation](../deployment.md) within existing clusters, adding minimal [components](../components/overview.md) to your setup. This facilitates straightforward deployments using your existing YAML configurations, Helm charts, or Kustomization, enabling native Kubernetes orchestration of your applications.

* **Remote attestation**: Contrast generates a concise attestation statement that verifies the identity, authenticity, and integrity of your deployment both internally and to external parties. This architecture ensures that updates or scaling of the application don't compromise the attestation’s validity.

* **Service mesh**: Contrast securely manages a Public Key Infrastructure (PKI) for your deployments, issues workload-specific certificates, and establishes transparent mutual TLS (mTLS) connections across pods. This is done by harnessing the [envoy proxy](https://www.envoyproxy.io/) to ensure secure communications within your Kubernetes cluster.

* **GPU support**: Contrast supports GPU integration, enabling the execution of AI workloads.  
