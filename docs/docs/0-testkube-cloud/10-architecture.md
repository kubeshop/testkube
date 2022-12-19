---
sidebar_position: 10
sidebar_label: Testkube architecture
---

## Multiple Testkube Agents

Main Testkube Cloud feature is to have insights into multiple Testkube Cloud Agents. 
You can look at your Kubernetes clusters from single dashboard. 


## Storing results

In Testkube standalone all results are stored in the users cluster, you need to be aware of MinIO and MongoDB. 
Testkube Cloud will make it easy for you, all data are stored in Testkube Cloud infrastructure.


## Testkube networking

To simplify networking connections Testkube Agent is able to create tunnel to Testkube Cloud clusters. The main 
idea of it is to allow Testkube Cloud to send commands which Testkube in Agent mode will manage. 

Testkube Agent after installing connects to Testkube Cloud, and starts listening for the commands. 
Additionally Agent is connecting to usual Testkube Cloud REST API.


