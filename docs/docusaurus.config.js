// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

import { themes as prismThemes } from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
async function createConfig() {
  return {
    title: 'Contrast',
    tagline: 'Contrast: Confidential Containers at scale',
    url: 'https://docs.edgeless.systems',
    baseUrl: '/contrast/',
    trailingSlash: false,
    onBrokenLinks: 'throw',
    onBrokenMarkdownLinks: 'throw',
    onBrokenAnchors: 'throw',
    favicon: 'img/favicon.ico',

    // GitHub pages deployment config.
    // If you aren't using GitHub pages, you don't need these.
    organizationName: 'edgelesssys', // Usually your GitHub org/user name.
    projectName: 'contrast', // Usually your repo name.
    deploymentBranch: 'gh-pages',

    // scripts
    scripts: [
      {
        src: '/contrast/gtagman.js',
        async: true,
        "data-cookieconsent": "ignore",
      },
    ],

    // Even if you don't use internalization, you can use this field to set useful
    // metadata like html lang. For example, if your site is Chinese, you may want
    // to replace "en" with "zh-Hans".
    i18n: {
      defaultLocale: 'en',
      locales: ['en'],
    },

    // mermaid
    markdown: {
      mermaid: true,
    },
    themes: ['@docusaurus/theme-mermaid'],

    presets: [
      [
        'classic',
        /** @type {import('@docusaurus/preset-classic').Options} */
        ({
          docs: {
            sidebarPath: require.resolve('./sidebars.js'),
            editUrl: 'https://github.com/edgelesssys/contrast/edit/main/docs',
            routeBasePath: "/"
          },
          blog: false,
          theme: {
            customCss: require.resolve('./src/css/custom.css'),
          },
        }),
      ],
    ],

    themeConfig:
      /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
      ({
        navbar: {
          hideOnScroll: false,
          logo: {
            alt: 'Contrast Logo',
            src: 'img/logos/contrast_icon.svg',
          },
          items: [
            {
              type: 'docsVersionDropdown',
              position: 'right',
            },
            {
              href: 'https://github.com/edgelesssys/contrast',
              position: 'right',
              className: 'header-github-link',
            },
          ],
        },
        colorMode: {
          defaultMode: 'light',
          disableSwitch: true,
          respectPrefersColorScheme: false,
        },
        announcementBar: {
          content:
            'Contrast is now open source! Check it out on <a target="_blank" rel="noopener noreferrer" href="https://github.com/edgelesssys/contrast">GitHub</a> and leave us a ⭐️ if you like it!',
          backgroundColor: '#E7E6E6'
        },
        footer: {
          style: 'dark',
          links: [
            {
              title: 'Community',
              items: [
                {
                  label: 'Newsletter',
                  href: 'https://www.edgeless.systems/#footer-id'
                },
              ],
            },
            {
              title: 'Social',
              items: [
                {
                  label: 'Blog',
                  href: 'https://www.edgeless.systems/blog/',
                },
                {
                  label: 'Twitter',
                  href: 'https://twitter.com/EdgelessSystems',
                },
                {
                  label: 'LinkedIn',
                  href: 'https://www.linkedin.com/company/edgeless-systems/',
                },
                {
                  label: 'Youtube',
                  href: 'https://www.youtube.com/channel/UCOOInN0sCv6icUesisYIDeA',
                },
              ],
            },
            {
              title: 'Company',
              items: [
                {
                  label: 'Imprint',
                  href: 'https://www.edgeless.systems/imprint/',
                },
                {
                  label: 'Privacy Policy',
                  href: 'https://www.edgeless.systems/privacy/',
                },
                {
                  html: `<a href="javascript: Cookiebot.renew()" class="footer__link-item">Cookie Settings</a>`
                },
                {
                  label: 'Contact Us',
                  href: 'https://www.edgeless.systems/contact-us/',
                },
              ],
            },
          ],
          copyright: `Copyright © ${new Date().getFullYear()} Edgeless Systems`,
        },
        prism: {
          theme: prismThemes.github,
          darkTheme: prismThemes.dracula,
          additionalLanguages: ['shell-session', 'bash', 'json', 'diff'],
        },
        mermaid: {
          theme: { light: 'base', dark: 'base' },
          options: {
            themeVariables: {
              // general
              fontFamily: '"Open Sans", sans-serif',
              primaryColor: '#90FF99', // edgeless green
              primaryTextColor: '#000000',
              secondaryColor: '#A5A5A5', // edgeless grey
              secondaryTextColor: '#000000',
              tertiaryColor: '#E7E6E6', // edgeless light grey
              tertiaryTextColor: '#000000',
              // flowchart
              clusterBorder: '#A5A5A5',
              clusterBkg: '#ffffff',
              edgeLabelBackground: '#ffffff',
              // sequence diagram
              activationBorderColor: '#000000',
              actorBorder: '#A5A5A5',
              actorFontFamily: '"Open Sans", sans-serif', // not released by mermaid yet
              noteBkgColor: '#8B04DD', // edgeless purple
              noteTextColor: '#ffffff',
            },
            startOnLoad: true,
          },
        },
      }),

    plugins: [
      [
        require.resolve("@cmfcmf/docusaurus-search-local"),
        {
          // whether to index docs pages
          indexDocs: true,

          // Whether to also index the titles of the parent categories in the sidebar of a doc page.
          // 0 disables this feature.
          // 1 indexes the direct parent category in the sidebar of a doc page
          // 2 indexes up to two nested parent categories of a doc page
          // 3...
          //
          // Do _not_ use Infinity, the value must be a JSON-serializable integer.
          indexDocSidebarParentCategories: 0,

          // whether to index blog pages
          indexBlog: false,

          // whether to index static pages
          // /404.html is never indexed
          indexPages: false,

          // language of your documentation, see next section
          language: "en",

          // setting this to "none" will prevent the default CSS to be included. The default CSS
          // comes from autocomplete-theme-classic, which you can read more about here:
          // https://www.algolia.com/doc/ui-libraries/autocomplete/api-reference/autocomplete-theme-classic/
          // When you want to overwrite CSS variables defined by the default theme, make sure to suffix your
          // overwrites with `!important`, because they might otherwise not be applied as expected. See the
          // following comment for more information: https://github.com/cmfcmf/docusaurus-search-local/issues/107#issuecomment-1119831938.
          style: undefined,

          // The maximum number of search results shown to the user. This does _not_ affect performance of
          // searches, but simply does not display additional search results that have been found.
          maxSearchResults: 8,

          // lunr.js-specific settings
          lunr: {
            // When indexing your documents, their content is split into "tokens".
            // Text entered into the search box is also tokenized.
            // This setting configures the separator used to determine where to split the text into tokens.
            // By default, it splits the text at whitespace and dashes.
            //
            // Note: Does not work for "ja" and "th" languages, since these use a different tokenizer.
            tokenizerSeparator: /[\s\-]+/,
            // https://lunrjs.com/guides/customising.html#similarity-tuning
            //
            // This parameter controls the importance given to the length of a document and its fields. This
            // value must be between 0 and 1, and by default it has a value of 0.75. Reducing this value
            // reduces the effect of different length documents on a term’s importance to that document.
            b: 0.75,
            // This controls how quickly the boost given by a common word reaches saturation. Increasing it
            // will slow down the rate of saturation and lower values result in quicker saturation. The
            // default value is 1.2. If the collection of documents being indexed have high occurrences
            // of words that are not covered by a stop word filter, these words can quickly dominate any
            // similarity calculation. In these cases, this value can be reduced to get more balanced results.
            k1: 1.2,
            // By default, we rank pages where the search term appears in the title higher than pages where
            // the search term appears in just the text. This is done by "boosting" title matches with a
            // higher value than content matches. The concrete boosting behavior can be controlled by changing
            // the following settings.
            titleBoost: 5,
            contentBoost: 1,
            tagsBoost: 3,
            parentCategoriesBoost: 2, // Only used when indexDocSidebarParentCategories > 0
          }
        },
      ]
    ]
  }
};

module.exports = createConfig;
