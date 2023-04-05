## Organizations management


To manage your organization settings click "Manage Organization" from organizations drop-down menu

![image](https://user-images.githubusercontent.com/30776/230005688-f88ae2f2-5570-4b25-80e5-ae524a384437.png)

You can also create new organiztion. 


## Organization settings

To edit your organization settings click one from the available options from menu on the left

### Environments

In environments section you can see list of your existing environments. 

![image](https://user-images.githubusercontent.com/30776/230006228-70275cca-1365-4d04-8387-725cf87c448b.png)

Green status colot mean that your agent is connected successfully. 

In case of a RED status you can try to debug the issues with the command below:

```sh
testkube agent debug
```

on your cluster where the given agent is installed



### Settings

In settings you can remove your organiztion. Keep in mind that this operation can't be rolled-back! 

![image](https://user-images.githubusercontent.com/30776/230007193-6d6004c1-57b3-4ea5-9c36-68aa4933ca57.png)



### Members

For each organization you can define who can access it and what kind on actions given member can do. 

![organiation-members](https://user-images.githubusercontent.com/30776/230007820-afbd49b1-b918-42ad-80de-a4d59714c2e6.png)


There are 4 types of organization members: 

* `owner` - has access to all environments and organization settings, also can access billing details
* `admin` - has access to all environments and organization settings.
* `biller` - has access to billing details only.
* `member` - has limited access to environments, access is defined by the roles assigned to given member. Member by default doesn't have any access, you need to [explicitly set it in given environment](testkube-cloud/environment-management). 



### API Tokens

Sometimes you need machine-to-machine authorization to e.g. run tests in CI pipelines, or just call particular actions from your services. 
Testkube offers API Tokens to resolve this issue. API Tokens have very similar roles like members. 

Each token can have also expiration date, you can set it for given time period or as "No expiration" (not recommended for production environments).
If token is not needed anymore yo ucan delete it from tokens list. 

API Tokens can have 2 roles: 

#### "admin" access to all environments

![image](https://user-images.githubusercontent.com/30776/230009462-3dee4b99-3bf4-4b5c-986d-806077b33281.png)

#### "member" limited access to environments, either limited access for environment actions 

![image](https://user-images.githubusercontent.com/30776/230010012-607b69da-24e8-4ec7-8888-f004759a1dd1.png)

For member organization role you should choose which environments you want to add to the created API Token, additionally for each 
environment roles should be choosen: 

![image](https://user-images.githubusercontent.com/30776/230010190-cacd1798-794f-466e-ac5c-d68801d23ed0.png)

We have 3 available roles for environment access: 
* `Read` - read only, you can only call get and list endpoints which not mutate data in any way
* `Run` - you can read and run - no changes to the environment.
* `Write` - you can change environments and run tests. 

### Usage & Billing

#### Upgrading testkube to `PRO` plan

![Zrzut ekranu 2023-04-5 o 09 31 43](https://user-images.githubusercontent.com/30776/230012570-7c1a67c9-77a5-4c02-903a-9f0fa93c9279.png)

#### `Free` plan usage 

All limits are calculated monthly. You have: 
- 600 executions 
- 2 environments
- 2GB artifacts storage
- 3 Users

![image](https://user-images.githubusercontent.com/30776/230013186-0f5c748d-58fb-4c9c-83df-5210c613ebaa.png)



#### `PRO` plan usage

`PRO` plan doesn't have any limits, you need to subscribe it and you'll get: 
- 5000 test executions
- **Unlimited** environments
- **128GB** of artifacts storage
- **25** users

If you need more - just use testkube - and you'll be charged for additional usage
(For prices details follow your subscription details by clicking "Manage subscription" button) 

![image](https://user-images.githubusercontent.com/30776/230013404-444eda20-04e5-4422-99ff-bfb05b4424ba.png)


