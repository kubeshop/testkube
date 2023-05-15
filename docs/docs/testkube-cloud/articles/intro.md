# Testkube Cloud

Testkube Cloud is the managed version of Testkube with the main purpose of:
- Orchestrating tests throughout multiple clusters 
- Managing different environments for testing (development, staging, production, etc.) 
- Enabling enterprise authentication and RBAC
- Simplifying test artifacts storage

## How it works? 

The way Testkube Cloud works is by installing adding an agent to the Testkube installation in your cluster, which then connects with Testkube's servers. This allows Testkube to offer these added functionalities while you can still benefit from Testkube's main feature of running your testing tools inside your cluster. 

## Getting started 

You can start using Testkube Cloud by either: 
- [**Migrating Testkube Open Source**](./migrating-testkube-oss.md) your existing Testkube Open Source instance to a Cloud instance
- Creating a fresh installation, using [cloud.testkube.io](https://cloud.testkube.io)