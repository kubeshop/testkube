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
  // By default, Docusaurus generates a sidebar from the docs folder structure
  tutorialSidebar: [
    {
      type: "category",
      label: "Overview",
      link: {
        type: "doc",
        id: "index",
      },
      items: [
        "overview/supported-tests",
        "overview/does-testkube-replace-cicd",
      ],
    },
    {
      type: "category",
      label: "Getting Started",
      link: {
        type: "doc",
        id: "getting-started/overview",
      },
      items: [
        "getting-started/installing-cli",
        "getting-started/installing-cluster-components",
        "getting-started/creating-first-test",
      ],
    },
    {
      type: "category",
      label: "Guides",
      items: [
        {
          type: "category",
          label: "Tests",
          items: [
            "guides/tests/tests-creating",
            "guides/tests/tests-running",
            "guides/tests/tests-getting-results",
            "guides/tests/tests-variables",
          ],
        },
        {
          type: "category",
          label: "Test Suites",
          items: [
            "guides/test-suites/testsuites-creating",
            "guides/test-suites/testsuites-running",
            "guides/test-suites/testsuites-getting-results",
          ],
        },
        {
          type: "category",
          label: "Exposing Testkube Dashboard",
          link: {
            type: "doc",
            id: "guides/exposing-testkube/overview",
          },
          items: ["guides/exposing-testkube/ingress-nginx"],
        },
        "guides/dashboard",
        "guides/secrets",
        "guides/scheduling",
        "guides/artifacts-storage",
        "guides/metrics",
        "guides/triggers",
        "guides/dependencies",
        "guides/common-issues",
      ],
    },
    {
      type: "category",
      label: "Test Types",
      link: {
        type: "generated-index",
        description: "Supported Test Types / Executors within Testkube",
      },
      items: [
        "test-types/executor-artillery",
        "test-types/executor-curl",
        "test-types/executor-cypress",
        "test-types/executor-ginkgo",
        "test-types/executor-gradle",
        "test-types/executor-jmeter",
        "test-types/executor-k6",
        "test-types/executor-kubepug",
        "test-types/executor-maven",
        "test-types/executor-playwright",
        "test-types/executor-postman",
        "test-types/executor-soapui",
        "test-types/executor-custom",
        "test-types/container-executor",
      ],
    },
    {
      type: "category",
      label: "Testkube Cloud",
      items: [
        "testkube-cloud/intro",
        "testkube-cloud/installing-agent",
        "testkube-cloud/architecture",
      ],
    },
    {
      type: "category",
      label: "Integrations",
      items: [
        "integrations/testkube-automation",
        "integrations/webhooks",
        "integrations/slack-integration",
        "integrations/generate-test-crds",
        {
          type: "category",
          label: "Authentication",
          items: [
            "integrations/authentication/oauth-cli",
            "integrations/authentication/oauth-ui",
          ],
        },
      ],
    },
    {
      type: "category",
      label: "Observability",
      items: ["observability/telemetry", "observability/logging"],
    },
    {
      type: "category",
      label: "Quick Links",
      items: ["quick-links/uninstall"],
    },
    {
      type: "category",
      label: "Reference",
      items: [
        {
          type: "doc",
          id: "reference/helm-chart",
          label: "Helm Chart",
        },
        "reference/openapi",
        "reference/architecture",
      ],
    },
    {
      type: "category",
      label: "Contributing",
      items: [
        "contributing/intro",
        {
          type: "category",
          label: "Development",
          items: [
            "contributing/development/development-crds",
            "contributing/development/developments",
          ],
        },
      ],
    },
  ],

  // But you can create a sidebar manually
  /*
  tutorialSidebar: [
    {
      type: 'category',
      label: 'Tutorial',
      items: ['hello'],
    },
  ],
   */
};

module.exports = sidebars;
