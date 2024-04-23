# Disable Secret Creation

Setting `testkube-api.disableSecretCreation` to `true` will prevent the creation of new secrets from the dashboard and the API.  
This setting hides the options for creating secrets, allowing only references to existing secrets to be used.

For instance, in areas where git credentials (such as username and token) would typically be entered, these inputs will also be hidden. Instead, users will see inputs designed for referencing existing secrets.

This change ensures that only pre-defined secrets can be utilized, enhancing security by limiting the ability to add new secrets dynamically.

[Back to Helm Chart reference](./helm-chart.md)
