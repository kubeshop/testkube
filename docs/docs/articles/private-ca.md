# Using a private CA

Installations which must serve Testkube endpoints or Git repositories using
certificates signed by a private CA should use this guide to make sure Testkube
components trust the private CA.

## Configurations

To get started we need to create a bundle containing all the CA certificates we
would like the installation to trust.

After that you will need to create a secret with the CA bundle under the
`ca.crt` key in the namespace(s) with the Helm releases both for the agent and
the enterprise control plane.

```
kubectl -n <namespace> create secret generic <secret name> \
    --from-file=ca.crt=<path to the file with the ca bundle>
```

Configure the following value in the `testkube-enterprise` chart:

```yaml
global:
    customCaSecretRef: <secret name>
```

Configure the following values in the `testkube` chart:

```yaml
global:
    testWorkflows:
        globalTemplate:
            enabled: true
            spec:
                pod:
                    volumes:
                        - name: testkube-enterprise-ca
                          secret:
                              secretName: <secret name>
                              defaultMode: 420
                container:
                    env:
                        - name: SSL_CERT_DIR
                          value: /etc/testkube/certs/
                        - name: GIT_SSL_CAINFO
                          value: /etc/testkube/certs/testkube-custom-ca.pem
                    volumeMounts:
                        - name: testkube-enterprise-ca
                          mountPath: /etc/testkube/certs/testkube-custom-ca.pem
                          subPath: ca.crt
testkube-api:
    cloud:
        tls:
            customCaSecretRef: <secret name>
```

### Pulling from Git repositories

If you would like to be able to pull Git data from repositories served both by
GitHub (or any other host) and your own Git servers which utilize private CA
signed certificates you will need to bundle the root CA certificates for those
hosts by concatenating them into one CA bundle.

### Prebuilt executors

Should work out of the box with `customCaSecretRef` set.

### Container executors

Pulling from Git repositories should work with `customCaSecretRef` set, but if
the execution workload must trust your private CA **you will need** to bake your
CAs into the right location inside your image.

### Workflows

With the global template specified above you should be able to both mount and
trust the CA bundle for pulling from Git and processing the results of the test
executions, but if the images specified by you as part of the workflow require
trust of your private-CA **you will need** to configure the container/pod in a
similar way as shown in the global template to mount the CA-bundle to the right
location and/or if necessary specify an environment variable.
