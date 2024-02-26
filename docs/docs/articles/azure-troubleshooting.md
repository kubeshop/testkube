# Azure DevOps Troubleshooting

## Testkube CLI and Git Integration issue

When integrating Testkube with Azure DevOps, a common issue that users might encounter involves the --git flags in the Testkube CLI. This problem manifests as the process becoming stuck without displaying any error messages, ultimately leading to a timeout. This document provides a solution to circumvent this issue, ensuring a smoother integration and execution of tests within Azure DevOps pipelines.

To avoid this issue, it is recommended to use the Git CLI directly for cloning the necessary repositories before executing Testkube CLI commands that reference the local copies of the test files or directories. This approach bypasses the complications associated with the --git flags in Testkube CLI within Azure DevOps environments.

### Example Workflow Adjustment

#### Before Adjustment (Issue Prone):
```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Test
  jobs:
  - job: RunTestkube
    steps:
      - task: SetupTestkube@1
      - script: |
          testkube create test --name test-name --test-content-type git-file --git-uri <git-repo> --git-path test-path
          testkube run test test-name
        displayName: Run Testkube Test
```

#### After Adjustment (Recommended Solution):
```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Test
  jobs:
  - job: RunTestkube
    steps:
      - task: SetupTestkube@1
      - script: |
          git clone <git-repo>
          testkube create test --name test-name -f test-path
          testkube run test test-name
        displayName: Run Testkube Test
```
