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
      items: ["articles/supported-tests", "articles/testkube-benefits"],
    },
    {
      type: "category",
      label: "Getting Started",
      link: {
        type: "doc",
        id: "articles/getting-started-overview",
      },
      items: [
        "articles/step1-installing-cli",
        "articles/step2-installing-cluster-components",
        "articles/step3-creating-first-test",
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
            "articles/creating-tests",
            "articles/running-tests",
            "articles/getting-tests-results",
            "articles/adding-tests-variables",
            "articles/adding-timeout",
          ],
        },
        {
          type: "category",
          label: "Test Suites",
          items: [
            "articles/creating-test-suites",
            "articles/running-test-suites",
            "articles/getting-test-suites-results",
          ],
        },
        {
          type: "category",
          label: "Testkube Dashboard",
          link: {
            type: "doc",
            id: "articles/testkube-dashboard",
          },
          items: [
            "articles/testkube-dashboard-explore",
            "articles/testkube-dashboard-general-settings",
            "articles/testkube-dashboard-api-endpoint",
          ],
        },
        "articles/adding-tests-secrets",
        "articles/scheduling-tests",
        "articles/test-triggers",
        "articles/webhooks",
        "articles/test-sources",
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
              label: "Authentication",
              items: ["articles/oauth-cli", "articles/oauth-dashboard"],
            },
            "articles/exposing-testkube-with-ingress-nginx",
            "articles/deploying-in-aws",
          ],
        },
        {
          type: "category",
          label: "CI/CD Integration",
          link: {
            type: "doc",
            id: "articles/cicd-overview",
          },
          items: [
            "articles/github-actions",
            "articles/run-tests-with-github-actions",
            "articles/testkube-cli-docker",
            {
              type: "category",
              label: "GitOps",
              link: {
                type: "doc",
                id: "articles/gitops-overview",
              },
              items: [
                {
                  type: "doc",
                  id: "articles/flux-integration",
                  label: "Flux",
                },
                {
                  type: "doc",
                  id: "articles/argocd-integration",
                  label: "ArgoCD",
                },
              ],
            },
          ],
        },
        "articles/cd-events",
        "articles/slack-integration",
        "articles/generate-test-crds",
        "articles/logging",
        "articles/uninstall",
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
        "test-types/executor-tracetest",
        "test-types/executor-zap",
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
        "testkube-cloud/articles/intro",
        "testkube-cloud/articles/installing-agent",
        "testkube-cloud/articles/transition-from-oss",
        "testkube-cloud/articles/organization-management",
        "testkube-cloud/articles/environment-management",
        "testkube-cloud/articles/managing-cli-context",
        "testkube-cloud/articles/architecture",
      ],
    },
    {
      type: "category",
      label: "Reference",
      items: [
        {
          type: "doc",
          id: "articles/helm-chart",
          label: "Helm Chart",
        },
        "articles/crds-reference",
        {
          type: "category",
          label: "CLI",
          items: [
            {
              type: "autogenerated",
              dirName: "cli",
            },
          ],
        },
        "openapi",
        "articles/metrics",
        "articles/artifacts-storage",
        "articles/testkube-dependencies",
        "articles/architecture",
        "articles/telemetry",
      ],
    },
    "articles/common-issues",
    {
      type: "category",
      label: "Contributing",
      items: [
        "articles/contributing",
        {
          type: "category",
          label: "Development",
          link: {
            type: "doc",
            id: "articles/development",
          },
          items: ["articles/crds"],
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
