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
      label: "Introduction",
      id: "intro",
    },
    {
      type: "category",
      label: "Getting started",
      collapsed: false,
      items: [
        {
          type: "doc",
          label: "Overview",
          id: "getting-started/overview",
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
      collapsed: false,
      items: [
        {
          type: "category",
          label: "Cluster setup",
          items: [
            {
              type: "doc",
              label: "Bare metal",
              id: "howto/cluster-setup/bare-metal",
            },
          ],
        },
        {
          type: "doc",
          label: "Install CLI",
          id: "howto/install-cli",
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
              label: "Add Coordinator to resources",
              id: "howto/workload-deployment/add-coordinator",
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
              label: "Connect to Coordinator",
              id: "howto/workload-deployment/connect-to-coordinator",
            },
            {
              type: "doc",
              label: "Set manifest",
              id: "howto/workload-deployment/set-manifest",
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
          label: "Set up encrypted volumes",
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
          label: "Registry authentication",
          id: "howto/registry-authentication",
        },
        {
          type: "doc",
          label: "Observability",
          id: "howto/observability",
        },
        {
          type: "doc",
          label: "Manifest update",
          id: "howto/manifest-update",
        },
        {
          type: "doc",
          label: "Coordinator high-availability",
          id: "howto/coordinator-ha",
        },
        {
          type: "doc",
          label: "Vault",
          id: "howto/vault",
        },
        {
          type: "doc",
          label: "Secure image store",
          id: "howto/secure-image-store",
        },
        {
          type: "doc",
          label: "Immutable deployments",
          id: "howto/immutable-deployments"
        },
        {
          type: "doc",
          label: "Troubleshooting",
          id: "howto/troubleshooting",
        },
      ],
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
          type: "category",
          label: "Components",
          items: [
            {
              type: "doc",
              label: "Runtime",
              id: "architecture/components/runtime",
            },
            {
              type: "doc",
              label: "Policies",
              id: "architecture/components/policies",
            },
            {
              type: "doc",
              label: "Initializer",
              id: "architecture/components/initializer",
            },
            {
              type: "doc",
              label: "Coordinator",
              id: "architecture/components/coordinator",
            },
            {
              type: "doc",
              label: "Service mesh",
              id: "architecture/components/service-mesh",
            },
          ],
        },
        {
          type: "category",
          label: "Attestation",
          items: [
            {
              type: "doc",
              label: "Overview",
              id: "architecture/attestation/overview",
            },
            {
              type: "doc",
              label: "AMD SEV-SNP",
              id: "architecture/attestation/amd-details",
            },
            {
              type: "doc",
              label: "Attested TLS",
              id: "architecture/attestation/atls",
            },
          ],
        },
        {
          type: "doc",
          label: "Secrets & recovery",
          id: "architecture/secrets",
        },
        {
          type: "doc",
          label: "K8s YAML elements",
          id: "architecture/k8s-yaml-elements",
        },
        {
          type: "doc",
          label: "Features & limitations",
          id: "architecture/features-limitations",
        },
        {
          type: "doc",
          label: "Telemetry & data collection",
          id: "architecture/telemetry",
        },
      ],
    },
  ],
};

module.exports = sidebars;
