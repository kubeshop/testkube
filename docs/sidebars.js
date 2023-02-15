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
      items: ["overview/faq"],
    },
    {
      type: "doc",
      id: "getting-started",
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
      label: "Using Testkube",
      items: [
        {
          type: "category",
          label: "Tests",
          items: [
            "using-testkube/tests/tests-creating",
            "using-testkube/tests/tests-running",
            "using-testkube/tests/tests-getting-results",
            "using-testkube/tests/tests-variables",
          ],
        },
        {
          type: "category",
          label: "Tests Suites",
          items: [
            "using-testkube/test-suites/testsuites-creating",
            "using-testkube/test-suites/testsuites-running",
            "using-testkube/test-suites/testsuites-getting-results",
          ],
        },
        "using-testkube/UI",
        "using-testkube/secrets",
        "using-testkube/scheduling",
        "using-testkube/artifacts-storage",
        "using-testkube/metrics",
        "using-testkube/triggers",
        "using-testkube/dependencies",
      ],
    },
    {
      type: "category",
      label: "Test Types",
      link: {
        type: 'generated-index',
        description: "Supported Test Types / Executors within Testkube"
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
        "test-types/executor-postman",
        "test-types/executor-soapui",
        "test-types/executor-custom",
        "test-types/container-executor",
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
      label: "Reference",
      items: [
        "reference/installation",
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
