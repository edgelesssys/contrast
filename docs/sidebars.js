/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    {
      type: "doc",
      label: "Introductions",
      id: "intro",
    },
    {
      type: "category",
      label: "Getting started",
      items: [
        {
          type: "doc",
          label: "Overview",
          id: "getting-started/overview",
        },
        {
          type: "doc",
          label: "Installation & setup",
          id: "getting-started/install",
        },
        {
          type: "doc",
          label: "Workload deployment",
          id: "getting-started/deployment",
        },
      ],
    },
    {
      type: "category",
      label: "How-to",
      items: [
        {
          type: "category",
          label: "Cluster setup",
          items: [
            {
              type: "doc",
              label: "AKS",
              id: "howto/cluster-setup/aks",
            },
            {
              type: "doc",
              label: "Bare metal",
              id: "howto/cluster-setup/bare-metal",
            },
          ],
        },
        {
          type: "category",
          label: "Workload deployment",
          items: [
            {
              type: "doc",
              label: "Deploy runtime",
              id: "howto/workload-deployment/runtime-deployment",
            },
            {
              type: "doc",
              label: "Prepare deployment files",
              id: "howto/workload-deployment/deployment-file-preparation",
            },
            {
              type: "doc",
              label: "Configure TLS",
              id: "howto/workload-deployment/TLS-configuration",
            },
            {
              type: "doc",
              label: "Enable GPU support",
              id: "howto/workload-deployment/GPU-configuration",
            },
            {
              type: "doc",
              label: "Generate annotations & manifest",
              id: "howto/workload-deployment/generate-annotations",
            },
            {
              type: "doc",
              label: "Deploy application",
              id: "howto/workload-deployment/deploy-application",
            },
            {
              type: "doc",
              label: "Deploy Conrast coordinator",
              id: "howto/workload-deployment/deploy-coordinator",
            },
            {
              type: "doc",
              label: "Verify deployment",
              id: "howto/workload-deployment/deployment-verification",
            },
            {
              type: "doc",
              label: "Communicate with workloads",
              id: "howto/workload-deployment/workload-communication",
            },
            {
              type: "doc",
              label: "Recover Contrast coordinator",
              id: "howto/workload-deployment/recover-coordinator",
            },
          ],
        },
        {
          type: "doc",
          label: "Setup encrypted volumes",
          id: "howto/encrypted-storage",
        },
        {
          type: "doc",
          label: "Hardening",
          id: "howto/hardening",
        },
        {
          type: "doc",
          label: "Logging",
          id: "howto/logging",
        },
        {
          type: "doc",
          label: "Observability",
          id: "howto/observability",
        },
        {
          type: "doc",
          label: "Recovery",
          id: "howto/recovery",
        },
        {
          type: "doc",
          label: "Manifest update",
          id: "howto/manifest-update",
        },
      ],
    },
    {
      type: "doc",
      label: "Troubleshooting",
      id: "troubleshooting",
    },
    {
      type: "doc",
      label: "Security",
      id: "security",
    },
    {
      type: "category",
      label: "Architecture",
      items: [
        {
          type: "doc",
          label: "Overview",
          id: "architecture/overview",
        },
        {
          type: "doc",
          label: "Components",
          id: "architecture/components",
        },
        {
          type: "doc",
          label: "Attestation",
          id: "architecture/attestation",
        },
      ],
    },
    {
      type: "category",
      label: "Old docs",
      items: [
        {
          type: "doc",
          label: "What is Contrast?",
          id: "old/intro",
        },
        {
          type: "category",
          label: "Basics",
          items: [
            {
              type: "doc",
              label: "Confidential Containers",
              id: "old/basics/confidential-containers",
            },
            {
              type: "doc",
              label: "Security benefits",
              id: "old/basics/security-benefits",
            },
            {
              type: "doc",
              label: "Features",
              id: "old/basics/features",
            },
          ],
        },
        {
          type: "category",
          label: "Getting started",
          items: [
            {
              type: "doc",
              label: "Install",
              id: "old/getting-started/install",
            },
            {
              type: "doc",
              label: "Cluster setup",
              id: "old/getting-started/cluster-setup",
            },
            {
              type: "doc",
              label: "Bare metal setup",
              id: "old/getting-started/bare-metal",
            },
          ],
        },
        {
          type: "category",
          label: "Examples",
          items: [
            {
              type: "doc",
              label: "Confidential emoji voting",
              id: "old/examples/emojivoto",
            },
            {
              type: "doc",
              label: "Encrypted volume mount",
              id: "old/examples/mysql",
            },
          ],
        },
        {
          type: "doc",
          label: "Workload deployment",
          id: "old/deployment",
        },
        {
          type: "doc",
          label: "Troubleshooting",
          id: "old/troubleshooting",
        },
        {
          type: "category",
          label: "Components",
          items: [
            {
              type: "doc",
              label: "Overview",
              id: "old/components/overview",
            },
            {
              type: "doc",
              label: "Runtime",
              id: "old/components/runtime",
            },
            {
              type: "doc",
              label: "Policies",
              id: "old/components/policies",
            },
            {
              type: "doc",
              label: "Service mesh",
              id: "old/components/service-mesh",
            },
          ],
        },
        {
          type: "category",
          label: "Architecture",
          items: [
            {
              type: "doc",
              label: "Attestation",
              id: "old/architecture/attestation",
            },
            {
              type: "doc",
              label: "Secrets & recovery",
              id: "old/architecture/secrets",
            },
            {
              type: "doc",
              label: "Certificate authority",
              id: "old/architecture/certificates",
            },
            {
              type: "doc",
              label: "Security considerations",
              id: "old/architecture/security-considerations",
            },
            {
              type: "doc",
              label: "Observability",
              id: "old/architecture/observability",
            },
            {
              type: "doc",
              label: "AMD SEV-SNP attestation",
              id: "old/architecture/snp",
            },
          ],
        },
        {
          type: "category",
          label: "How-To",
          items: [
            {
              type: "doc",
              label: "Registry authentication",
              id: "old/howto/registry-authentication",
            },
          ],
        },
        {
          type: "doc",
          label: "Planned features and limitations",
          id: "old/features-limitations",
        },
        {
          type: "category",
          label: "About",
          items: [
            {
              type: "doc",
              label: "Telemetry",
              id: "old/about/telemetry",
            },
          ],
        },
      ],
    },
  ],
};

module.exports = sidebars;
