---
sidebar_position: 10
sidebar_label: Testkube architecture
---

## Multiple Testkube Agents

Main Testkube Cloud feature is to have insights into multiple Testkube Cloud Agents. 
You can look at your Kubernetes clusters from single dashboard and easily switch between different Testkube clusters. 


![multiple clusters](https://user-images.githubusercontent.com/30776/208391158-a42d1f56-950f-48c3-bcfb-2768054b4704.jpeg)


## Storing results

In Testkube standalone all results are stored in the users cluster, you need to be aware of MinIO and MongoDB. 
Testkube Cloud will make it easy for you, all data are stored in Testkube Cloud infrastructure so you don't need to worry about backups, .


## Testkube networking

To simplify networking connections Testkube Agent is able to create connection to Testkube Cloud clusters, Agent is registering itself into 
Testkube Cloud and shows as new environment. 
The main idea of it is to allow Testkube Cloud to send commands which Testkube in Agent mode will manage. Connection is done 
from Testkube Agent to Testkube Cloud.

Testkube Agent after installing connects to Testkube Cloud, and starts listening for the commands. 
Additionally Agent is connecting to usual Testkube Cloud REST API.


![network](https://user-images.githubusercontent.com/30776/208391192-6f04ce7a-2c8a-4892-bc01-3a3b04cd3ddc.jpeg)

Testkube Agent is connecting to `https://api.testkube.io` on port `8088` for HTTPS connection and on port `8089` for GRPC connection.  

