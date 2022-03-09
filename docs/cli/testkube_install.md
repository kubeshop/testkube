# Testkube Install

Install the Helm chart registry in the current kubectl context.

## **Synopsis**

The install can be configured with the use of particular flags passed to the install command:

```
testkube install [flags]
```

## **Options**

```
      --chart string   Chart name (default "kubeshop/testkube").
  -h, --help           Help for install.
      --name string    Installation name (default "testkube").
      --no-dashboard   Don't install dashboard.
      --no-jetstack    Don't install Jetstack.
      --no-minio       Don't install MinIO.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled   Enable analytics (default "true").
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy").
  -s, --namespace string    Kubernetes namespace (default "testkube").
  -v, --verbose             Show additional debug messages.
```

## **SEE ALSO**

* [Testkube](testkube.md)	 - The testkube entrypoint for plugins.

