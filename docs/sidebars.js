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
      items: ["overview/supported-tests", "overview/testkube-benefits"],
    },
    {
      type: "category",
      label: "Getting Started",
      link: {
        type: "doc",
        id: "getting-started/index",
      },
      items: [
        "getting-started/step1-installing-cli",
        "getting-started/step2-installing-cluster-components",
        "getting-started/step3-creating-first-test",
      ],
    },
    {
      type: "category",
      label: "Concepts",
      items: [
        {
          type: "category",
          label: "Tests",
          items: [
            "concepts/tests/tests-creating",
            "concepts/tests/tests-running",
            "concepts/tests/tests-getting-results",
            "concepts/tests/tests-variables",
          ],
        },
        {
          type: "category",
          label: "Test Suites",
          items: [
            "concepts/test-suites/testsuites-creating",
            "concepts/test-suites/testsuites-running",
            "concepts/test-suites/testsuites-getting-results",
          ],
        },
        "concepts/dashboard",
        "concepts/secrets",
        "concepts/scheduling",
        "concepts/artifacts-storage",
        "concepts/metrics",
        "concepts/triggers",
        "concepts/dependencies",
        "concepts/common-issues",
      ],
    },
    {
      type: "category",
      label: "Guides",
      items: [
        {
          type: "category",
          label: "Getting to Production",
          items: [
            {
              type: "category",
              label: "Exposing Testkube Dashboard",
              link: {
                type: "doc",
                id: "guides/going-to-production/exposing-testkube/overview",
              },
              items: [
                "guides/going-to-production/exposing-testkube/ingress-nginx",
              ],
            },
            {
              type: "category",
              label: "Authentication",
              items: [
                "guides/going-to-production/authentication/oauth-cli",
                "guides/going-to-production/authentication/oauth-ui",
              ],
            },
            "guides/going-to-production/aws",
          ],
        },
        {
          type: "category",
          label: "CI/CD Integration",
          link: {
            type: "doc",
            id: "guides/cicd/index",
          },
          items: [
            "guides/cicd/github-actions",
            {
              type: "category",
              label: "GitOps",
              link: {
                type: "doc",
                id: "guides/cicd/gitops/index",
              },
              items: [
                {
                  type: "doc",
                  id: "guides/cicd/gitops/flux",
                  label: "Flux",
                },
                {
                  type: "doc",
                  id: "guides/cicd/gitops/argocd",
                  label: "ArgoCD",
                },
              ],
            },
          ],
        },
        "guides/webhooks",
        "guides/slack-integration",
        "guides/generate-test-crds",
        "guides/logging",
        "guides/uninstall",
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
        "test-types/prebuilt-executor",
        "test-types/container-executor",
      ],
    },
    {
      type: "html",
      value: "<hr />",
    },
    {
      type: "category",
      label: "Testkube Cloud",
      items: [
        "testkube-cloud/intro",
        "testkube-cloud/installing-agent",
        "testkube-cloud/transition-from-oss",
        "testkube-cloud/organization-management",
        "testkube-cloud/environment-management",
        "testkube-cloud/managing-cli-context",
        "testkube-cloud/architecture",
      ],
    },
    {
      type: "category",
      label: "Reference",
      items: [
        {
          type: "category",
          label: "CLI",
          items: [
            {
              type: "autogenerated",
              dirName: "reference/cli",
            },
          ],
        },
        {
          type: "doc",
          id: "reference/helm-chart",
          label: "Helm Chart",
        },
        "reference/openapi",
        "reference/architecture",
        "reference/telemetry",
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
          link: {
            type: "doc",
            id: "contributing/development/index",
          },
          items: ["contributing/development/crds"],
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
