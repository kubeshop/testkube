// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("./src/themes/prism-testkube-light");
const darkCodeTheme = require("./src/themes/prism-testkube-dark");
const redirects = require("./redirects");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Testkube Documentation",
  tagline:
    "Your somewhat opinionated and friendly Kubernetes testing framework",
  url: "https://docs.testkube.io",
  baseUrl: "/",
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
  scripts: [
    {
      src: "https://app.usercentrics.eu/browser-ui/latest/loader.js",
      id: "usercentrics-cmp",
      "data-settings-id": "WQ2gSqnsK",
      async: true,
    },
  ],
  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      {
        docs: {
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.js"),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl: "https://github.com/kubeshop/testkube/tree/develop/docs",
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
        gtag: {
          trackingID: "G-G7HWN1EDK5",
        },
        googleTagManager: {
          containerId: "GTM-PQK4DKN",
        },
      },
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
    {
      navbar: {
        logo: {
          alt: "Testkube",
          src: "img/logo.large.bright.svg",
          srcDark: "img/logo.large.svg",
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

      announcementBar: {
        id: "join-testkube-pro",
        content: '<strong>New to Testkube?</strong> Unleash the power of cloud native testing in Kubernetes with Testkube. <a id="sign_cloud_docs" href="https://app.testkube.io/?utm_source=docs" target="_blank">Get Started ></a>',
        backgroundColor: "#8951ff",
        textColor: "#fff",
      },

      footer: {
        style: "dark",
        links: [
          {
            title: "Community",
            items: [
              {
                label: "Slack",
                href: "https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email",
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

        // Optional: path for search page that enabled by default (`false` to disable it)
        searchPagePath: false,

        //... other Algolia params
      },
      colorMode: {
        defaultMode: "dark",
        disableSwitch: false,
        respectPrefersColorScheme: true,
      },
    },
  plugins: [
    [
      "@docusaurus/plugin-client-redirects",
      {
        redirects: redirects,
        createRedirects(existingPath) {
          if (existingPath.includes("/cli")) {
            // Redirect from /cli-reference and /reference/cli (old links) to /cli
            return [existingPath.replace('/cli', "/cli-reference"), existingPath.replace('/cli', "/reference/cli")];
          }

          return undefined; // Return a falsy value: no redirect created
        },
      },
    ],
  ],
};

module.exports = config;
