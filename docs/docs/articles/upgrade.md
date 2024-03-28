# Upgrade Testkube

Upgrading Testkube will upgrade the cluster components. 

There are two ways to upgrade Testkube: 

## Using Helm

:::note
By default, the namespace for the installation will be `testkube`.
To upgrade the `testkube` chart if it was installed into the default namespace:

```sh
helm upgrade my-testkube kubeshop/testkube
```

And for a namespace other than `testkube`:

```sh
helm upgrade --namespace namespace_name my-testkube kubeshop/testkube
```
:::

## Using Testkube's CLI

```bash
testkube upgrade
```