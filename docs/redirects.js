const redirects = [
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
    from: ["/executor-curl", "/test-types/curl"],
    to: "/test-types/executor-curl",
  },
  {
    from: ["/test-types/executor-custom", "/executor-custom"],
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
    from: "/testsuites-running",
    to: "/articles/running-test-suites",
  },
  {
    from: [
      "/testsuites-creating",
      "/concepts/test-suites/testsuites-creating",
      "/using-testkube/test-suites/testsuites-creating",
    ],
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
    from: "/guides/test-suites/testsuites-getting-results",
    to: "/articles/getting-test-suites-results",
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
    from: ["/integrations/testkube-automation", "/testkube-automation"],
    to: "/articles/cicd-overview",
  },
  {
    from: [
      "/guides/tests/tests-creating",
      "/category/tests",
      "/concepts/tests/tests-creating",
      "/tests-creating",
      "/using-testkube/tests/tests-creating",
    ],
    to: "/articles/creating-tests",
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
    from: "/overview/supported-tests",
    to: "/articles/supported-tests",
  },
  {
    from: "/overview/testkube-benefits",
    to: "/articles/testkube-benefits",
  },
  {
    from: [
      "/getting-started/index",
      "/getting-started/installation",
      "/getting-started",
      "/integrations",
      "/installing",
      "/articles/getting-started-overview",
      "/getting-started/step2-installing-cluster-components",
      "/articles/step2-installing-cluster-components",
      "/getting-started/step3-creating-first-test",
      "/articles/step3-creating-first-test",
    ],
    to: "/articles/getting-started",
  },
  {
    from: [
      "/cli/installation",
      "/getting-started/installing-cli",
      "/articles/step1-installing-cli",
      "/getting-started/step1-installing-cli",
    ],
    to: "/articles/install-cli",
  },
  {
    from: "/concepts/tests/tests-running",
    to: "/articles/running-tests",
  },
  {
    from: "/concepts/tests/tests-getting-results",
    to: "/articles/getting-tests-results",
  },
  {
    from: [
      "/concepts/tests/tests-variables",
      "/tests-variables",
      "/using-testkube/tests/tests-variables",
    ],
    to: "/articles/adding-tests-variables",
  },
  {
    from: "/concepts/test-suites/testsuites-running",
    to: "/articles/running-test-suites",
  },
  {
    from: "/concepts/test-suites/testsuites-getting-results",
    to: "/articles/getting-test-suites-results",
  },
  {
    from: "/concepts/dashboard",
    to: "/articles/testkube-dashboard",
  },
  {
    from: "/concepts/secrets",
    to: "/articles/adding-tests-secrets",
  },
  {
    from: "/concepts/scheduling",
    to: "/articles/scheduling-tests",
  },
  {
    from: ["/concepts/artifacts", "/using-testkube/artifacts"],
    to: "/articles/artifacts",
  },
  {
    from: "/concepts/metrics",
    to: "/articles/metrics",
  },
  {
    from: ["/concepts/triggers", "/using-testkube/triggers"],
    to: "/articles/test-triggers",
  },
  {
    from: "/concepts/dependencies",
    to: "/articles/testkube-dependencies",
  },
  {
    from: "/concepts/common-issues",
    to: "/articles/common-issues",
  },
  {
    from: "/concepts/test-sources",
    to: "/articles/test-sources",
  },
  {
    from: "/concepts/test-executions",
    to: "/articles/test-executions",
  },
  {
    from: [
      "/guides/going-to-production/exposing-testkube/overview",
      "/guides/exposing-testkube/overview",
      "/articles/exposing-testkube",
    ],
    to: "/articles/testkube-dashboard",
  },
  {
    from: [
      "/guides/going-to-production/exposing-testkube/ingress-nginx",
      "/guides/exposing-testkube/ingress-nginx",
      "/articles/exposing-testkube-with-ingress-nginx",
    ],
    to: "/articles/testkube-dashboard",
  },
  {
    from: [
      "/guides/going-to-production/authentication/oauth-cli",
      "/guides/authentication/oauth-cli",
    ],
    to: "/articles/oauth-cli",
  },
  {
    from: [
      "/guides/going-to-production/authentication/oauth-ui",
      "/articles/oauth-dashboard",
    ],
    to: "/articles/testkube-dashboard",
  },
  {
    from: "/guides/going-to-production/aws",
    to: "/articles/deploying-in-aws",
  },
  {
    from: "/guides/cicd",
    to: "/articles/cicd-overview",
  },
  {
    from: "/guides/cicd/github-actions",
    to: "/articles/github-actions",
  },
  {
    from: "/guides/cicd/gitops",
    to: "/articles/gitops-overview",
  },
  {
    from: "/guides/cicd/gitops/flux",
    to: "/articles/flux-integration",
  },
  {
    from: "/guides/cicd/gitops/argocd",
    to: "/articles/argocd-integration",
  },
  {
    from: "/guides/webhooks",
    to: "/articles/webhooks",
  },
  {
    from: "/guides/slack-integration",
    to: "/articles/slack-integration",
  },
  {
    from: "/guides/generate-test-crds",
    to: "/articles/generate-test-crds",
  },
  {
    from: "/guides/logging",
    to: "/articles/logging",
  },
  {
    from: "/guides/uninstall",
    to: "/articles/uninstall",
  },
  {
    from: "/guides/templates",
    to: "/articles/templates",
  },
  {
    from: [
      "/testkube-cloud/intro",
      "/testkube-cloud",
      "/testkube-cloud/articles/intro",
      "/testkube-pro/intro",
    ],
    to: "/testkube-pro/articles/intro",
  },
  {
    from: [
      "/testkube-cloud/installing-agent",
      "/testkube-cloud/articles/installing-agent",
      "/testkube-pro/installing-agent",
    ],
    to: "/testkube-pro/articles/installing-agent",
  },
  {
    from: [
      "/testkube-cloud/transition-from-oss",
      "/testkube-cloud/articles/transition-from-oss",
      "/testkube-pro/transition-from-oss",
    ],
    to: "/testkube-pro/articles/transition-from-oss",
  },
  {
    from: [
      "/testkube-cloud/organization-management",
      "/testkube-cloud/articles/organization-management",
      "/testkube-pro/organization-management",
    ],
    to: "/testkube-pro/articles/organization-management",
  },
  {
    from: [
      "/testkube-cloud/environment-management",
      "/testkube-cloud/articles/environment-management",
      "/testkube-pro/environment-management",
    ],
    to: "/testkube-pro/articles/environment-management",
  },
  {
    from: [
      "/testkube-cloud/managing-cli-context",
      "/testkube-cloud/articles/managing-cli-context",
      "/testkube-pro/managing-cli-context",
    ],
    to: "/testkube-pro/articles/managing-cli-context",
  },
  {
    from: [
      "/testkube-cloud/architecture",
      "/testkube-cloud/articles/architecture",
      "/testkube-pro/architecture",
    ],
    to: "/testkube-pro/articles/architecture",
  },
  {
    from: [
      "/testkube-cloud/articles/running-parallel-tests-with-test-suite",
      "/testkube-pro/running-parallel-tests-with-test-suite",
    ],
    to: "/testkube-pro/articles/running-parallel-tests-with-test-suite",
  },
  {
    from: [
      "/testkube-cloud/articles/AI-test-insights",
      "/testkube-pro/AI-test-insights",
    ],
    to: "/testkube-pro/articles/AI-test-insights",
  },
  {
    from: [
      "/testkube-cloud/articles/status-pages",
      "/testkube-pro/status-pages",
    ],
    to: "/testkube-pro/articles/status-pages",
  },
  {
    from: [
      "/testkube-cloud/articles/cached-results",
      "/testkube-pro/cached-results",
    ],
    to: "/testkube-pro/articles/cached-results",
  },
  {
    from: "/reference/helm-chart",
    to: "/articles/helm-chart",
  },
  {
    from: "/reference/openapi",
    to: "/openapi",
  },
  {
    from: "/reference/architecture",
    to: "/articles/architecture",
  },
  {
    from: "/reference/telemetry",
    to: "/articles/telemetry",
  },
  {
    from: ["/contributing", "/contributing/contributing"],
    to: "/articles/contributing",
  },
  {
    from: "/contributing/development",
    to: "/articles/development",
  },
  {
    from: [
      "/contributing/development/crds",
      "/contributing/development/development-crds/",
    ],
    to: "/articles/crds",
  },
  {
    from: "/articles/operator-api-reference",
    to: "/articles/crds-reference",
  },
  {
    from: "/guides/upgrade",
    to: "/articles/upgrade",
  },
  {
    from: "/testkube-enterprise/articles/testkube-enterprise",
    to: "/testkube-pro-on-prem/articles/testkube-pro-on-prem",
  },
  {
    from: "/testkube-enterprise/articles/usage-guide",
    to: "/testkube-pro-on-prem/articles/usage-guide",
  },
  {
    from: "/testkube-enterprise/articles/migrating-from-oss-to-pro",
    to: "/testkube-pro-on-prem/articles/migrating-from-oss-to-pro",
  },
  {
    from: "/testkube-pro/articles/installing-agent",
    to: "/articles/install/advanced-multi-cluster",
  },
  {
    from: "/testkube-pro/articles/transition-from-oss",
    to: "/articles/install/advanced-multi-cluster",
  },
  {
    from: [
      "/testkube-enterprise/articles/auth",
      "/testkube-pro-on-prem/articles/auth",
    ],
    to: "/articles/install/auth",
  },
  {
    from: "/testkube-pro-on-prem/articles/testkube-pro-on-prem",
    to: "/articles/install/quickstart-install",
  },
  {
    from: "/testkube-pro-on-prem/articles/usage-guide",
    to: "/articles/install/install-with-helm",
  },
  {
    from: "/testkube-pro-on-prem/articles/migrating-from-oss-to-pro",
    to: "/articles/migrate-from-oss",
  },
  {
    from: "/articles/testkube-oss",
    to: "/articles/install/install-oss",
  },
];

module.exports = redirects;
