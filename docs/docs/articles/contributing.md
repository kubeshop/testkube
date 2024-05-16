# Contributing to Testkube Core Open Source

If you are new to the open source community, use this guide to start contributing to projects:
<https://github.com/firstcontributions/first-contributions>.

Checkout the [development document](./development.md) for more details about how to develop and run testkube on your machine.

## General Guidance for Contributing to a Testkube Project

Anyone is welcome and encouraged to help with Testkube development; much opportunity for enhancement exists.

We would like to limit technical debt from the beginning, so we have defined simple rules when adding code into Testkube repo.

### For Go Programming Language (Golang) Based Components

- Always use gofmt.
- Follow Golang good practices ([proverbs](https://go-proverbs.github.io/)) in your code.
- Testing is your friend. We will target 80% CC in our code.
- Use clean names and don't break basic design patterns and rules.

### For Infrastructure/Kubernetes Based Components

- Put in comments for non-obvious decisions.
- Use current Helm/Kubernetes versions.