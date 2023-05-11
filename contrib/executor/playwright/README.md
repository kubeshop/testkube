![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to Testkube Playwright Executor

Testkube Playwright Executor is the test executor for [Testkube](https://testkube.io).  
[Playwright](https://playwright.dev/) is a framework for Web Testing and Automation.

# Issues and enchancements

Please follow the main [Testkube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

## Details

If you use HTML Reporter. please disable auto open reporter in `playwright.config.js`.
```
reporter: [
  ['html', { open: 'never' }]
],
```
Otherwise, the test will not automatically terminate when the test fails.

## Architecture

- TODO add architecture diagrams

## API

Playwright executor implements [testkube OpenAPI for executors](https://docs.testkube.io/openapi#tag/executor) (look at executor tag)
