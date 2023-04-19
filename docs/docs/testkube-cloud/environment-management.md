# Environment management

Environment in Testkube is related to the Testkube agent, who is responsible for sending
test insight to Testkube Cloud, and for managing your Kubernetes related cluster resources.


## Creating new environment

You can create new environment from "Environments" drop down in header section of Testkube Cloud UI. 

![env-drop-down](https://user-images.githubusercontent.com/30776/230015851-eae48d9e-e634-4771-be2f-6d28e42bb55b.png)

Fro installation instruction follow [Testkube Agent Installation](installing-agent.md)

## Changing environment settings

![env-settings](https://user-images.githubusercontent.com/30776/230016969-a38e0915-ae4b-426a-a844-bb646ed85bdc.png)


On "General" tab you can see environment informatio like
* Connection state 
* Agent name
* Agent version - In case of new Testkube Agent version available you'll be noticed here to upgrade
* Testkube CLI context command - to configure your testkube CLI with cloud context

You can also delete given environment (be careful this action can't be rolled-back!)

![env-settings](https://user-images.githubusercontent.com/30776/230017592-160a0a5e-370f-4efe-9317-daedfad364b3.png)


## Managing environment members

Keep in mind that all organiazation `admin` users can access all environments.

To add new organization user with member role use "Members" tab.

![adding-new-menmber](https://user-images.githubusercontent.com/30776/230018272-50507361-12eb-47ea-8649-015392b69eea.png)

You can choose one from the roles for user: 

* `Read`: Has read access only to all entities in an environment, test results, artifacts, logs, etc...
* `Run`: Has Read + Can trigger Test/suite executions.
* `Write`: Has Run + Can make changes to environment tests, triggers, webhooks, etc...
* `Admin`: Has write + is allowed to invite and change other collaborator roles.




