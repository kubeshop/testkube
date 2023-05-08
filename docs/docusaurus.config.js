// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("./src/themes/prism-testkube-light");
const darkCodeTheme = require("./src/themes/prism-testkube-dark");

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
          editUrl: "https://github.com/kubeshop/testkube/docs",
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
          src: "img/logo.large.svg",
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
    },
  plugins: [
    [
      "@docusaurus/plugin-client-redirects",
      {
        redirects: [
          // /docs/oldDoc -> /docs/newDoc
          {
            from: "/executor-cypress",
            to: "/test-types/executor-cypress",
          },
          {
            from: "/executor-postman",
            to: "/test-types/executor-postman",
          },
          {
            from: "/executor-soapui",
            to: "/test-types/executor-soapui",
          },
          {
            from: "/executor-k6",
            to: "/test-types/executor-k6",
          },
          {
            from: "/executor-jmeter",
            to: "/test-types/executor-jmeter",
          },
          {
            from: "/executor-kubepug",
            to: "/test-types/executor-kubepug",
          },
          {
            from: "/executor-artillery",
            to: "/test-types/executor-artillery",
          },
          {
            from: "/executor-maven",
            to: "/test-types/executor-maven",
          },
          {
            from: "/executor-gradle",
            to: "/test-types/executor-gradle",
          },
          {
            from: "/executor-ginkgo",
            to: "/test-types/executor-ginkgo",
          },
          {
            from: "/executor-curl",
            to: "/test-types/executor-curl",
          },
          {
            from: "/test-types/executor-custom",
            to: "/test-types/container-executor",
          },
          {
            from: "/UI",
            to: "/articles/testkube-dashboard",
          },
          {
            from: "/tests-running",
            to: "/articles/running-tests",
          },
          {
            from: "/tests-creating",
            to: "/articles/creating-tests",
          },
          {
            from: "/tests-variables",
            to: "/articles/adding-tests-variables",
          },
          {
            from: "/testsuites-running",
            to: "/articles/running-test-suites",
          },
          {
            from: "/testsuites-creating",
            to: "/articles/creating-test-suites",
          },
          {
            from: "/helm-charts",
            to: "/articles/helm-chart",
          },
          {
            from: "/telemetry",
            to: "/articles/telemetry",
          },
          {
            from: "/installing",
            to: "/articles/getting-started-overview",
          },
          {
            from: "/guides/test-suites/testsuites-getting-results",
            to: "/articles/getting-test-suites-results",
          },
          {
            from: "/category/tests",
            to: "/articles/creating-tests",
          },
          {
            from: "/using-testkube/UI",
            to: "/articles/testkube-dashboard",
          },
          {
            from: "/FAQ",
            to: "/articles/common-issues",
          },
          {
            from: "/integrations/testkube-automation",
            to: "/articles/cicd-overview",
          },
          {
            from: "/guides/tests/tests-creating",
            to: "/articles/creating-tests",
          },
          {
            from: "/guides/exposing-testkube/ingress-nginx",
            to: "/articles/exposing-testkube-with-ingress-nginx",
          },
          {
            from: "/guides/exposing-testkube/overview",
            to: "/articles/exposing-testkube",
          },
          {
            from: "/architecture",
            to: "/articles/architecture",
          },
          {
            from: "/integrations/slack-integration",
            to: "/articles/slack-integration",
          },
          {
            from: "/integrations",
            to: "/articles/getting-started-overview",
          },
        ],
        createRedirects(existingPath) {
          if (existingPath.includes("/cli")) {
            // Redirect from /cli-reference to /reference/cli
            return [existingPath.replace("/cli", "/cli-reference")];
          }

          if (existingPath.includes("/articles")) {
            // Redirect from /using-testkube to /articles
            return [existingPath.replace("/articles", "/using-testkube")];
          }

          if (
            existingPath.includes("/guides/going-to-production/authentication")
          ) {
            return [
              existingPath.replace(
                "/guides/going-to-production/authentication",
                "/authentication"
              ),
            ];
          }
          return undefined; // Return a falsy value: no redirect created
        },
      },
    ],
  ],
};

module.exports = config;
