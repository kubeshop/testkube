# Uninstall Testkube

Uninstalling Testkube will remove the cluster components, but the namespace will not be deleted. 

There are two ways for uninstalling Testkube: 

## Using Helm

```bash
helm delete testkube
```

## Using Testkube's CLI

```bash
testkube purge
```