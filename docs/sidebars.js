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
      type: 'category',
      label: 'What is Contrast?',
      collapsed: false,
      link: {
        type: 'doc',
        id: 'intro'
      },
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
      link: {
        type: 'doc',
        id: 'getting-started/index'
      },
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
      ]
    },
    {
      type: 'category',
      label: 'Examples',
      link: {
        type: 'doc',
        id: 'examples/index'
      },
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
      type: 'category',
      label: 'Components',
      link: {
        type: 'doc',
        id: 'components/index'
      },
      items: [
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
          label: 'Service Mesh',
          id: 'components/service-mesh',
        },
      ]
    },
    {
      type: 'category',
      label: 'Architecture',
      link: {
        type: 'doc',
        id: 'architecture/index'
      },
      items: [
        {
          type: 'doc',
          label: 'Attestation',
          id: 'architecture/attestation',
        },
        {
          type: 'doc',
          label: 'Certificate authority',
          id: 'architecture/certificates',
        },
      ]
    },
    {
      type: 'doc',
      label: 'Known limitations',
      id: 'known-limitations',
    },
    {
      type: 'category',
      label: 'About',
      link: {
        type: 'doc',
        id: 'about/index'
      },
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
