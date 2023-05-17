# Connecting from the CLI

To use Testkube CLI to connect to your Testkube Cloud install you will need to set the CLI Context. For that you will need a Testkube Cloud token.

The token you create can have different roles associated with it for fine-grained control:

* **Admin** access rights (access to all environments):

![Admin Token](../../img/admin-token.png)

<!-- ![admin-token](https://user-images.githubusercontent.com/30776/229772185-01f1e466-b04d-4c6d-9d5c-e4464d651177.png) -->

* Specific role for selected environments:

![Roles for Token](../../img/roles-for-token.png)

<!-- ![roles-for-token](https://user-images.githubusercontent.com/30776/229772310-64bda85d-57a8-47b7-a68b-2625089724f8.png) -->



When the token is created, you're ready to change the Testkube CLI context: 

![Setting Context](../../img/setting-context.png)

<!-- ![setting-context](https://user-images.githubusercontent.com/30776/229771159-4415aa74-70bb-4684-9511-449d0779b483.png) -->


# Connecting Using `kubeconfig` Context

If you want to connect to your Testkube instance directly (like you would do with `kubectl`), set the CLI Context to be `kubeconfig`-based: 

```sh 
testkube set context --kubeconfig
```

