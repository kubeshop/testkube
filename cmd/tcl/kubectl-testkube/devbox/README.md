# Development Box - TCL Licensed

This utility is used to help with development of the Agent features (like Test Workflows). 

## How it works

* It takes current Testkube CLI credentials and create development environment inside
* It deploys the Agent into the current cluster
  * Test Triggers are disabled
  * Webhooks are disabled
  * Legacy Tests and Test Suites are disabled
  * It's not using Helm Chart, so default templates are not available
* For live changes, it deploys Interceptor and Binary Storage into the current cluster
  * Binary Storage stores latest binaries for the Agent, Toolkit and Init Process
  * Binary Storage is optimized for patching binaries with incremental builds (to avoid sending the whole binary, when only small part is changed)
  * Interceptor loads the Toolkit and Init Process from the Object Storage into every Test Workflow Execution pod

## Usage

* Login to Testkube CLI, like `testkube login`
  * For local development Testkube Enterprise (Skaffold), consider `testkube login localhost:8099`
  * It's worth to create alias for that in own `.bashrc` or `.bash_profile`
  * It's worth to pass a devbox name, like `-n dawid`, so it's not using random name
* For OSS version - run with `--oss` parameter

The CLI will print a dashboard link for the selected environment.

## Why?

It's a fast way to get live changes during the development:
* initial deployment takes up to 60 seconds
* continuous deployments take 1-10 seconds (depending on changes and network bandwidth)
* the Execution performance is not much worse (it's just running single container before, that is only fetching up to 100MB from local Object Storage)

## Parameters

Most important parameters are `-n, --name` for devbox static name,
and `-s, --fssync` for synchronising Test Workflow and Test Workflow Template CRDs from the file system.

```shell
Usage:
  testkube devbox [flags]

Aliases:
  devbox, dev

Flags:
  -n, --name string            devbox name (default "1730107481990508000")
  -s, --fssync strings           synchronise resources at paths
  -o, --open                   open dashboard in browser
  -O, --oss                    run open source version
      --agent-image string     base agent image (default "kubeshop/testkube-api-server:latest")
      --init-image string      base init image (default "kubeshop/testkube-tw-init:latest")
      --toolkit-image string   base toolkit image (default "kubeshop/testkube-tw-toolkit:latest")
```

## Example

```shell
# Initialize alias
tk() {
  cd ~/projects/testkube
  go run cmd/kubectl-testkube/main.go $@
}

# Select the proper cluster to deploy the devbox
kubectx cloud-dev

# Run development box, synchronising all the Test Workflows from 'test' directory in Testkube repository
tk devbox -n dawid -s test
```
