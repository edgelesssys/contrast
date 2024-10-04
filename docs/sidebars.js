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
      type: 'doc',
      label: 'What is Contrast?',
      id: 'intro',
    },
    {
      type: 'category',
      label: 'Basics',
      collapsed: false,
      items: [
        {
          type: 'doc',
          label: 'Confidential Containers',
          id: 'basics/confidential-containers',
        },
        {
          type: 'doc',
          label: 'Security benefits',
          id: 'basics/security-benefits',
        },
        {
          type: 'doc',
          label: 'Features',
          id: 'basics/features',
        },
      ]
    },
    {
      type: 'category',
      label: 'Getting started',
      collapsed: false,
      items: [
        {
          type: 'doc',
          label: 'Install',
          id: 'getting-started/install',
        },
        {
          type: 'doc',
          label: 'Cluster setup',
          id: 'getting-started/cluster-setup',
        },
        {
          type: 'doc',
          label: 'Bare metal setup',
          id: 'getting-started/bare-metal',
        },
      ]
    },
    {
      type: 'category',
      label: 'Examples',
      items: [
        {
          type: 'doc',
          label: 'Confidential emoji voting',
          id: 'examples/emojivoto'
        },
      ]
    },
    {
      type: 'doc',
      label: 'Workload deployment',
      id: 'deployment',
    },
    {
      type: 'doc',
      label: 'Troubleshooting',
      id: 'troubleshooting',
    },
    {
      type: 'category',
      label: 'Components',
      items: [
        {
          type: 'doc',
          label: 'Overview',
          id: 'components/overview',
        },
        {
          type: 'doc',
          label: 'Runtime',
          id: 'components/runtime',
        },
        {
          type: 'doc',
          label: 'Policies',
          id: 'components/policies',
        },
        {
          type: 'doc',
          label: 'Service mesh',
          id: 'components/service-mesh',
        },
      ]
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        {
          type: 'doc',
          label: 'Attestation',
          id: 'architecture/attestation',
        },
        {
          type: 'doc',
          label: 'Secrets & recovery',
          id: 'architecture/secrets',
        },
        {
          type: 'doc',
          label: 'Certificate authority',
          id: 'architecture/certificates',
        },
        {
          type: 'doc',
          label: 'Security considerations',
          id: 'architecture/security-considerations',
        },
        {
          type: 'doc',
          label: 'Observability',
          id: 'architecture/observability',
        },
      ]
    },
    {
      type: 'doc',
      label: 'Planned features and limitations',
      id: 'features-limitations',
    },
    {
      type: 'category',
      label: 'About',
      items: [
        {
          type: 'doc',
          label: 'Telemetry',
          id: 'about/telemetry',
        },
      ]
    },
  ],
};

module.exports = sidebars;
