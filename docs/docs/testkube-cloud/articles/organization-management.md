# Organizations Management


To manage your organization settings click "Manage Organization" from organizations drop-down menu:

![Organization Settings](../../img/org-settings.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230005688-f88ae2f2-5570-4b25-80e5-ae524a384437.png) -->

You can also create new organization. 


## Organization Settings

To edit your organization settings, click an organization from the available options from menu on the left.

### Environments

In the environments section you can see the list of your existing environments.

![Existing Environments](../../img/existing-environments.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230006228-70275cca-1365-4d04-8387-725cf87c448b.png) -->

GREEN status means that your agent is connected successfully. 

In the case of a RED status, you can try to debug the issues with the command below:

```sh
testkube agent debug
```

Run this on your cluster where the given agent is installed.



### Settings

In settings you can remove your organiztion. Keep in mind that this operation can't be rolled-back!

![Delete Organization](../../img/delete-org.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230007193-6d6004c1-57b3-4ea5-9c36-68aa4933ca57.png) -->



### Members

For each organization you can define who has access and what kind of actions each member can use. 

![Organization Members](../../img/org-members.png)

<!-- ![organiation-members](https://user-images.githubusercontent.com/30776/230007820-afbd49b1-b918-42ad-80de-a4d59714c2e6.png)-->


There are 4 types of organization members: 

* `owner` - Has access to all environments and organization settings, also can access billing details.
* `admin` - Has access to all environments and organization settings.
* `biller` - Has access to billing details only.
* `member` - Has limited access to environments, access is defined by the roles assigned to given member. Member by default doesn't have any access, you need to [explicitly set it in the given environment](environment-management.md). 



### API Tokens

Sometimes you need machine-to-machine authorization to run tests in CI pipelines or  call particular actions from your services. 
Testkube offers API Tokens to resolve this issue. API Tokens have very similar roles like members. 

Each token can have also expiration date, you can set it for given time period or as "No expiration" (not recommended for production environments).
If token is not needed anymore you can delete it from the tokens list. 

API Tokens can have 2 roles: 

#### "admin" - access to all environments

![Admin Role](../../img/admin-roles.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230009462-3dee4b99-3bf4-4b5c-986d-806077b33281.png) -->

#### "member" - limited access to environments or limited access for environment actions 

![Member Role](../../img/member-role.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230010012-607b69da-24e8-4ec7-8888-f004759a1dd1.png) -->

For the member organization role, you should choose which environments you want to add to the created API Token, additionally, role should be chosen for each 
environment: 

![Environment Role](../../img/environment-role.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230010190-cacd1798-794f-466e-ac5c-d68801d23ed0.png) -->

We have 3 available roles for environment access: 
* `Read` - Read only, you can only call get and list endpoints which not mutate data in any way.
* `Run` - Access to Read and Run but no changes to the environment.
* `Write` - You can change environments and run tests. 

### Usage & Billing

#### Upgrading Testkube to the `PRO` Plan

![PRO Plan Billing](../../img/pro-plan-billing.png)

<!--![Zrzut ekranu 2023-04-5 o 09 31 43](https://user-images.githubusercontent.com/30776/230012570-7c1a67c9-77a5-4c02-903a-9f0fa93c9279.png)-->

#### `Free` Plan Usage 

All limits are calculated monthly. On the 'Free' plan, you have: 
- 600 executions 
- 2 environments
- 2GB artifacts storage
- 3 Users

![Free Plan](../../img/free-plan.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230013186-0f5c748d-58fb-4c9c-83df-5210c613ebaa.png) -->



#### `PRO` Plan Usage

Subscribing to the `PRO` plan provides: 
- 5000 test executions
- **Unlimited** environments
- **128GB** of artifacts storage
- **25** users

If you need more - just use Testkube - and you'll be charged for additional usage.
For pricing details, visit your subscription by clicking the "Manage subscription" button. 

![Manage Subcriptions](../../img/manage-subscriptions.png)

<!-- ![image](https://user-images.githubusercontent.com/30776/230013404-444eda20-04e5-4422-99ff-bfb05b4424ba.png) -->


