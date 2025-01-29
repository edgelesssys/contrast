"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[7502],{27305:e=>{e.exports=JSON.parse('{"version":{"pluginId":"default","version":"1.3","label":"1.3","banner":null,"badge":true,"noIndex":false,"className":"docs-version-1.3","isLast":true,"docsSidebars":{"docs":[{"type":"link","label":"What is Contrast?","href":"/contrast/pr-preview/pr-1195/","docId":"intro","unlisted":false},{"type":"category","label":"Basics","collapsed":false,"items":[{"type":"link","label":"Confidential Containers","href":"/contrast/pr-preview/pr-1195/basics/confidential-containers","docId":"basics/confidential-containers","unlisted":false},{"type":"link","label":"Security benefits","href":"/contrast/pr-preview/pr-1195/basics/security-benefits","docId":"basics/security-benefits","unlisted":false},{"type":"link","label":"Features","href":"/contrast/pr-preview/pr-1195/basics/features","docId":"basics/features","unlisted":false}],"collapsible":true},{"type":"category","label":"Getting started","collapsed":false,"items":[{"type":"link","label":"Install","href":"/contrast/pr-preview/pr-1195/getting-started/install","docId":"getting-started/install","unlisted":false},{"type":"link","label":"Cluster setup","href":"/contrast/pr-preview/pr-1195/getting-started/cluster-setup","docId":"getting-started/cluster-setup","unlisted":false},{"type":"link","label":"Bare metal setup","href":"/contrast/pr-preview/pr-1195/getting-started/bare-metal","docId":"getting-started/bare-metal","unlisted":false}],"collapsible":true},{"type":"category","label":"Examples","items":[{"type":"link","label":"Confidential emoji voting","href":"/contrast/pr-preview/pr-1195/examples/emojivoto","docId":"examples/emojivoto","unlisted":false},{"type":"link","label":"Encrypted volume mount","href":"/contrast/pr-preview/pr-1195/examples/mysql","docId":"examples/mysql","unlisted":false}],"collapsed":true,"collapsible":true},{"type":"link","label":"Workload deployment","href":"/contrast/pr-preview/pr-1195/deployment","docId":"deployment","unlisted":false},{"type":"link","label":"Troubleshooting","href":"/contrast/pr-preview/pr-1195/troubleshooting","docId":"troubleshooting","unlisted":false},{"type":"category","label":"Components","items":[{"type":"link","label":"Overview","href":"/contrast/pr-preview/pr-1195/components/overview","docId":"components/overview","unlisted":false},{"type":"link","label":"Runtime","href":"/contrast/pr-preview/pr-1195/components/runtime","docId":"components/runtime","unlisted":false},{"type":"link","label":"Policies","href":"/contrast/pr-preview/pr-1195/components/policies","docId":"components/policies","unlisted":false},{"type":"link","label":"Service mesh","href":"/contrast/pr-preview/pr-1195/components/service-mesh","docId":"components/service-mesh","unlisted":false}],"collapsed":true,"collapsible":true},{"type":"category","label":"Architecture","items":[{"type":"link","label":"Attestation","href":"/contrast/pr-preview/pr-1195/architecture/attestation","docId":"architecture/attestation","unlisted":false},{"type":"link","label":"Secrets & recovery","href":"/contrast/pr-preview/pr-1195/architecture/secrets","docId":"architecture/secrets","unlisted":false},{"type":"link","label":"Certificate authority","href":"/contrast/pr-preview/pr-1195/architecture/certificates","docId":"architecture/certificates","unlisted":false},{"type":"link","label":"Security considerations","href":"/contrast/pr-preview/pr-1195/architecture/security-considerations","docId":"architecture/security-considerations","unlisted":false},{"type":"link","label":"Observability","href":"/contrast/pr-preview/pr-1195/architecture/observability","docId":"architecture/observability","unlisted":false}],"collapsed":true,"collapsible":true},{"type":"link","label":"Planned features and limitations","href":"/contrast/pr-preview/pr-1195/features-limitations","docId":"features-limitations","unlisted":false},{"type":"category","label":"About","items":[{"type":"link","label":"Telemetry","href":"/contrast/pr-preview/pr-1195/about/telemetry","docId":"about/telemetry","unlisted":false}],"collapsed":true,"collapsible":true}]},"docs":{"about/telemetry":{"id":"about/telemetry","title":"CLI telemetry","description":"The Contrast CLI sends telemetry data to Edgeless Systems when you use CLI commands.","sidebar":"docs"},"architecture/attestation":{"id":"architecture/attestation","title":"Attestation in Contrast","description":"This document describes the attestation architecture of Contrast, adhering to the definitions of Remote ATtestation procedureS (RATS) in RFC 9334.","sidebar":"docs"},"architecture/certificates":{"id":"architecture/certificates","title":"Certificate authority","description":"The Coordinator acts as a certificate authority (CA) for the workloads","sidebar":"docs"},"architecture/observability":{"id":"architecture/observability","title":"Observability","description":"The Contrast Coordinator can expose metrics in the","sidebar":"docs"},"architecture/secrets":{"id":"architecture/secrets","title":"Secrets & recovery","description":"When the Coordinator is configured with the initial manifest, it generates a random secret seed.","sidebar":"docs"},"architecture/security-considerations":{"id":"architecture/security-considerations","title":"Security Considerations","description":"Contrast ensures application integrity and provides secure means of communication and bootstrapping (see security benefits).","sidebar":"docs"},"basics/confidential-containers":{"id":"basics/confidential-containers","title":"Confidential Containers","description":"Contrast uses some building blocks from Confidential Containers (CoCo), a CNCF Sandbox project that aims to standardize confidential computing at the pod level.","sidebar":"docs"},"basics/features":{"id":"basics/features","title":"Product features","description":"Contrast simplifies the deployment and management of Confidential Containers, offering optimal data security for your workloads while integrating seamlessly with your existing Kubernetes environment.","sidebar":"docs"},"basics/security-benefits":{"id":"basics/security-benefits","title":"Contrast security overview","description":"This document outlines the security measures of Contrast and its capability to counter various threats.","sidebar":"docs"},"components/overview":{"id":"components/overview","title":"Components","description":"Contrast is composed of several key components that work together to manage and scale confidential containers effectively within Kubernetes environments.","sidebar":"docs"},"components/policies":{"id":"components/policies","title":"Policies","description":"Kata runtime policies are an integral part of the Confidential Containers preview on AKS.","sidebar":"docs"},"components/runtime":{"id":"components/runtime","title":"Contrast Runtime","description":"The Contrast runtime is responsible for starting pods as confidential virtual machines.","sidebar":"docs"},"components/service-mesh":{"id":"components/service-mesh","title":"Service mesh","description":"The Contrast service mesh secures the communication of the workload by automatically","sidebar":"docs"},"deployment":{"id":"deployment","title":"Workload deployment","description":"The following instructions will guide you through the process of making an existing Kubernetes deployment","sidebar":"docs"},"examples/emojivoto":{"id":"examples/emojivoto","title":"Confidential emoji voting","description":"screenshot of the emojivoto UI","sidebar":"docs"},"examples/mysql":{"id":"examples/mysql","title":"Encrypted volume mount","description":"This tutorial guides you through deploying a simple application with an","sidebar":"docs"},"features-limitations":{"id":"features-limitations","title":"Planned features and limitations","description":"This section lists planned features and current limitations of Contrast.","sidebar":"docs"},"getting-started/bare-metal":{"id":"getting-started/bare-metal","title":"Prepare a bare-metal instance","description":"Hardware and firmware setup","sidebar":"docs"},"getting-started/cluster-setup":{"id":"getting-started/cluster-setup","title":"Create a cluster","description":"Prerequisites","sidebar":"docs"},"getting-started/install":{"id":"getting-started/install","title":"Installation","description":"Download the Contrast CLI from the latest release:","sidebar":"docs"},"intro":{"id":"intro","title":"Contrast","description":"Welcome to the documentation of Contrast! Contrast runs confidential container deployments on Kubernetes at scale.","sidebar":"docs"},"troubleshooting":{"id":"troubleshooting","title":"Troubleshooting","description":"This section contains information on how to debug your Contrast deployment.","sidebar":"docs"}}}}')}}]);