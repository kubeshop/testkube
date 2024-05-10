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
        "articles/supported-tests",
        "articles/testkube-benefits",
        "articles/open-source-or-pro",
      ],
    },
    {
      type: "doc",
      label: "Getting Started",
      id: "articles/getting-started",
    },
    {
      type: "category",
      label: "Installing",
      items: [
        { type: "doc", id: "articles/install/overview" },
        { type: "doc", id: "articles/install/cli" },
        { type: "doc", id: "articles/install/quickstart-install" },
        { type: "doc", id: "articles/install/reference-architectures" },
        { type: "doc", id: "articles/install/install-with-cli" },
        { type: "doc", id: "articles/install/install-with-helm" },
        { type: "doc", id: "articles/install/advanced-install" },
        { type: "doc", id: "articles/install/quickstart-no-k8s" },
        { type: "doc", id: "articles/install/uninstall" },
        { type: "doc", id: "articles/install/faq" },
      ],
    },
    {
      type: "category",
      label: "Concepts",
      items: [
        {
          type: "category",
          label: "Test Workflows (beta)",
          items: [
            "articles/test-workflows",
            "articles/test-workflows-creating",
            "articles/test-workflow-templates",
            "articles/test-workflows-examples-basics",
            "articles/test-workflows-examples-configuration",
            "articles/test-workflows-examples-expressions",
            "articles/test-workflows-examples-templates",
          ],
        },
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
          ],
        },
        "articles/adding-tests-secrets",
        "articles/scheduling-tests",
        "articles/test-triggers",
        "articles/webhooks",
        "articles/test-sources",
        "articles/test-executions",
        "articles/templates",
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
              items: ["articles/oauth-cli"],
            },
            "articles/deploying-in-aws",
            "articles/deploying-from-private-registries",
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
            {
              type: "category",
              label: "Github Actions",
              link: {
                type: "doc",
                id: "articles/github-actions",
              },
              items: [
                {
                  type: "doc",
                  id: "articles/run-tests-with-github-actions",
                  label: "Migrate from testkube-run-action",
                },
              ],
            },
            "articles/gitlab",
            "articles/jenkins",
            "articles/jenkins-ui",
            "articles/azure",
            "articles/circleci",
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
        "articles/creating-first-test",
        "articles/cd-events",
        "articles/slack-integration",
        "articles/generate-test-crds",
        "articles/logging",
        "articles/install-cli",
        "articles/uninstall",
        "articles/upgrade",
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
        "test-types/executor-kubent",
        "test-types/executor-maven",
        "test-types/executor-playwright",
        "test-types/executor-postman",
        "test-types/executor-pytest",
        "test-types/executor-soapui",
        "test-types/executor-tracetest",
        "test-types/executor-zap",
        "test-types/prebuilt-executor",
        "test-types/container-executor",
        "test-types/executor-distributed-jmeter",
      ],
    },
    {
      type: "html",
      value: "<hr />",
    },
    {
      type: "category",
      label: "Testkube Pro",
      items: [
        "testkube-pro/articles/intro",
        "testkube-pro/articles/installing-agent",
        "testkube-pro/articles/transition-from-oss",
        "testkube-pro/articles/organization-management",
        "testkube-pro/articles/environment-management",
        "testkube-pro/articles/managing-cli-context",
        "testkube-pro/articles/architecture",
        "testkube-pro/articles/running-parallel-tests-with-test-suite",
        "testkube-pro/articles/AI-test-insights",
        "testkube-pro/articles/status-pages",
        "testkube-pro/articles/cached-results",
        "testkube-pro/articles/log-highlighting",
      ],
    },
    {
      type: "category",
      label: "Testkube Pro On-Prem",
      items: [
        "testkube-pro-on-prem/articles/testkube-pro-on-prem",
        "testkube-pro-on-prem/articles/usage-guide",
        "testkube-pro-on-prem/articles/auth",
        "testkube-pro-on-prem/articles/migrating-from-oss-to-pro",
      ],
    },
    "articles/testkube-oss",
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
        "articles/artifacts",
        "articles/testkube-dependencies",
        "articles/architecture",
        "articles/telemetry",
        {
          type: 'link',
          label: 'OpenAPI Specification',
          href: 'https://api.testkube.io/openapi',
        },
      ],
    },
    "articles/common-issues",
    "articles/deprecations",
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
    {
      type: "category",
      label: "FAQs",
      items: ["articles/testkube-licensing-FAQ"],
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
