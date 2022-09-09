// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("./src/themes/prism-testkube-light");
const darkCodeTheme = require("./src/themes/prism-testkube-dark");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Testkube Documentation",
  tagline:
    "Your somewhat opinionated and friendly Kubernetes testing framework",
  url: "https://testkube.kubeshop.io",
  baseUrl: "/testkube/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img/logo.svg",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "kubeshop", // Usually your GitHub org/user name.
  projectName: "testkube", // Usually your repo name.

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.js"),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl: "https://github.com/kubeshop/testkube/docs",
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
        googleAnalytics: {
          trackingID: "UA-204665550-6",
        },
      }),
    ],
    [
      "redocusaurus",
      {
        // Plugin Options for loading OpenAPI files
        specs: [
          {
            spec: "https://raw.githubusercontent.com/kubeshop/testkube/main/api/v1/testkube.yaml",
            route: "/openapi",
          },
        ],
        theme: {
          primaryColor: "#818cf8",
        },
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        logo: {
          alt: "Testkube",
          src: "img/logo.svg",
          href: "/",
        },
        items: [
          {
            type: "html",
            position: "right",
            value: `<iframe src="https://ghbtns.com/github-btn.html?user=kubeshop&repo=testkube&type=star&count=true&color=dark" style='margin-top: 6px' frameborder="0" scrolling="0" width="110" height="20" title="GitHub"></iframe>`,
          },
          {
            type: "search",
            position: "left",
          },
        ],
      },

      footer: {
        style: "dark",
        links: [
          {
            title: "Community",
            items: [
              {
                label: "Discord",
                href: "https://discord.com/invite/6zupCZFQbe",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/Testkube_io",
              },
              {
                label: "GitHub",
                href: "https://github.com/kubeshop/testkube",
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Kubeshop.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
      algolia: {
        // The application ID provided by Algolia
        appId: "QRREOKFLDE",

        // Public API key: it is safe to commit it
        apiKey: "97a76158bf582346aa0c2605cbc593f6",
        indexName: "testkube",

        // Optional: see doc section below
        contextualSearch: false,

        // Optional: Specify domains where the navigation should occur through window.location instead on history.push. Useful when our Algolia config crawls multiple documentation sites and we want to navigate with window.location.href to them.
        // externalUrlRegex: "external\\.com|domain\\.com",

        // Optional: Algolia search parameters
        searchParameters: {},

        // Optional: path for search page that enabled by default (`false` to disable it)
        searchPagePath: "search",

        //... other Algolia params
      },
      colorMode: {
        defaultMode: "dark",
        disableSwitch: false,
        respectPrefersColorScheme: false,
      },
    }),
  plugins: [
    [
      "@docusaurus/plugin-client-redirects",
      {
        redirects: [
          // /docs/oldDoc -> /docs/newDoc
          {
            from: "/testkube/executor-cypress",
            to: "/testkube/test-types/executor-cypress",
          },
          {
            from: "/testkube/executor-postman",
            to: "/testkube/test-types/executor-postman",
          },
          {
            from: "/testkube/executor-soapui",
            to: "/testkube/test-types/executor-soapui",
          },
          {
            from: "/testkube/executor-k6",
            to: "/testkube/test-types/executor-k6",
          },
          {
            from: "/testkube/executor-kubepug",
            to: "/testkube/test-types/executor-kubepug",
          },
          {
            from: "/testkube/executor-artillery",
            to: "/testkube/test-types/executor-artillery",
          },
          {
            from: "/testkube/executor-maven",
            to: "/testkube/test-types/executor-maven",
          },
          {
            from: "/testkube/executor-gradle",
            to: "/testkube/test-types/executor-gradle",
          },
          {
            from: "/testkube/executor-ginkgo",
            to: "/testkube/test-types/executor-ginkgo",
          },
          {
            from: "/testkube/executor-curl",
            to: "/testkube/test-types/executor-curl",
          },
          {
            from: "/testkube/executor-custom",
            to: "/testkube/test-types/executor-custom",
          },
          {
            from: "/testkube/UI",
            to: "/testkube/using-testkube/UI",
          },
          {
            from: "/testkube/tests-running",
            to: "/testkube/using-testkube/tests/tests-running",
          },
          {
            from: "/testkube/tests-creating",
            to: "/testkube/using-testkube/tests/tests-creating",
          },
          {
            from: "/testkube/tests-variables",
            to: "/testkube/using-testkube/tests/tests-variables",
          },
          {
            from: "/testkube/testsuites-running",
            to: "/testkube/using-testkube/test-suites/testsuites-running",
          },
          {
            from: "/testkube/testsuites-creating",
            to: "/testkube/using-testkube/test-suites/testsuites-creating",
          },
        ],
      },
    ],
  ],
};

module.exports = config;
