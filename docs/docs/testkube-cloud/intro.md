# Intro

Testkube Cloud is the managed version of Testkube with the main purpose of orchestrating multiple clusters. 
All test results and test artifacts are stored into Testkube Cloud internal data storages. Testkube cloud 
will provide you with additional tests insights and is able to limit access for your users only to a subset 
of environments.

Testkube Cloud is in Alpha phase - so feel free to give us feedback! 


## Testkube Cloud Agent - Installation Manual

Testkube Cloud is able to connect to Testkube Agents. Testkube Agent is the Testkube engine for managing test runs into your cluster.
It is also responsible for getting insight into Testkube resources stored in the cluster.

Testkube Agent opens a networking connection into Testkube Cloud API, which is always active with the main purpose of listening for Testkube Cloud commands.

Your existing Open Source Testkube installation can be converted into a Testkube Cloud agent; data will be passed and managed by 
Testkube servers (Coming Soon!)

## Installing the Agent

Please follow the [install steps](installing-agent.md) to get started using the Testkube Agent.
