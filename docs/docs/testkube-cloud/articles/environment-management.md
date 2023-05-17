# Environment Management

Environment in Testkube is related to the Testkube agent, which is responsible for sending
test insights to Testkube Cloud and for managing your Kubernetes related cluster resources.


## Creating a New Environment

You can create a new environment from the "Environments" drop down in the header section of the Testkube Cloud UI. 

![env-drop-down](../../img/env-drop-down.png)

<!-- ![env-drop-down](https://user-images.githubusercontent.com/30776/230015851-eae48d9e-e634-4771-be2f-6d28e42bb55b.png)-->

For installation instructions, follow [Testkube Agent Installation](installing-agent.md)

## Changing Environment Settings

![env-settings](../../img/env-settings.png)

<!-- ![env-settings](https://user-images.githubusercontent.com/30776/230016969-a38e0915-ae4b-426a-a844-bb646ed85bdc.png) -->


On the "General" tab, you can see environment information:
* Connection state 
* Agent name
* Agent version - If a new Testkube Agent version is available, you'll be prompted to upgrade.
* Testkube CLI context command - To configure your Testkube CLI with cloud context.

You can also delete a given environment (be careful, this action can't be rolled-back!)

![env-information](../../img/env-information.png)

<!-- ![env-settings](https://user-images.githubusercontent.com/30776/230017592-160a0a5e-370f-4efe-9317-daedfad364b3.png) -->


## Managing Environment Member Roles

Keep in mind that all organization `admin` users can access all environments.

To add new organization users with member role use the "Members" tab.

![adding-new-member](../../img/adding-new-member.png)

<!-- ![adding-new-menmber](https://user-images.githubusercontent.com/30776/230018272-50507361-12eb-47ea-8649-015392b69eea.png) -->

You can choose from one of the following roles for a user: 

* `Read`: Has Read access only to all entities in an environment, test results, artifacts, logs, etc...
* `Run`: Has Read access and can trigger Test/ Test Suite executions.
* `Write`: Has Run access and can make changes to environment tests, triggers, webhooks, etc...
* `Admin`: Has Write access and is allowed to invite and change other collaborator roles.




