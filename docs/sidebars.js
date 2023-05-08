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
        "articles/testkube-dashboard",
        "articles/adding-tests-secrets",
        "articles/scheduling-tests",
        "articles/artifacts-storage",
        "articles/metrics",
        "articles/test-triggers",
        "articles/testkube-dependencies",
        "articles/common-issues",
        "articles/test-sources"
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
                id: "articles/exposing-testkube",
              },
              items: [
                "articles/exposing-testkube-with-ingress-nginx",
              ],
            },
            {
              type: "category",
              label: "Authentication",
              items: [
                "articles/oauth-cli",
                "articles/oauth-dashboard",
              ],
            },
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
        "articles/webhooks",
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
          type: "category",
          label: "CLI",
          items: [
            {
              type: "autogenerated",
              dirName: "cli",
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
