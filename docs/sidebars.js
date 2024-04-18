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
        {
          type: 'doc',
          label: 'First steps',
          id: 'getting-started/first-steps',
        }
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
          label: 'Service Mesh',
          id: 'components/service-mesh',
        },
        {
          type: 'doc',
          label: 'Pod VM image',
          id: 'components/podvm-image',
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
          type: 'category',
          label: 'Attestation',
          link: {
            type: 'generated-index',
          },
          items: [
            {
              type: 'doc',
              label: 'Hardware',
              id: 'architecture/attestation/hardware',
            },
            {
              type: 'doc',
              label: 'Pod VM',
              id: 'architecture/attestation/pod-vm',
            },
            {
              type: 'doc',
              label: 'Runtime policies',
              id: 'architecture/attestation/runtime-policies',
            },
            {
              type: 'doc',
              label: 'Manifest',
              id: 'architecture/attestation/manifest',
            },
            {
              type: 'doc',
              label: 'Coordinator',
              id: 'architecture/attestation/coordinator',
            },
          ]
        },
        {
          type: 'category',
          label: 'Certificates and Identities',
          link: {
            type: 'generated-index',
          },
          items: [
            {
              type: 'doc',
              label: 'PKI',
              id: 'architecture/certificates-and-identities/pki',
            },
          ]
        },
      ]
    },
  ],
};

module.exports = sidebars;
