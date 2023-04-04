# Migration from Testkube OSS

To migrate Testkube OSS to Cloud you need to install Testkube in Cloud Agent mode. Testkube Cloud Agent is the Testkube engine for managing test runs into your cluster. It sends data to Testkubes Cloud Servers. It's main responsibility is to manage test workloads and to get insight into Testkube resources stored in the cluster.


## Installing the Agent

Please follow the [install steps](installing-agent.md) to get started using the Testkube Agent.

## Migrating Testkube Resources

Currently there is no automatic migration tool for existing Testkube OSS resources. But we have plan for it in incoming releases.


## Changing the Testkube context to cloud one

To set context to could one, testkube CLI tool need to have access, so first you'll need to create API token with 
valid access rights.

You can create token:

* with "admin" access rights (access to all environments) 

![admin-token](https://user-images.githubusercontent.com/30776/229772185-01f1e466-b04d-4c6d-9d5c-e4464d651177.png)

* with particular role for given environments

![roles-for-token](https://user-images.githubusercontent.com/30776/229772310-64bda85d-57a8-47b7-a68b-2625089724f8.png)



Now when your token is there you're ready to change testkube CLI context: 

![setting-context](https://user-images.githubusercontent.com/30776/229771159-4415aa74-70bb-4684-9511-449d0779b483.png)

