#!/bin/sh

if [ -n "$KUBECONFIG_FILE" ]; then
	echo "copying KUBECONFIG_FILE to /tmp/kubeconfig/config and exporting KUBECONFIG"
        mkdir -p /tmp/kubeconfig
	echo $KUBECONFIG_FILE > /tmp/kubeconfig/config
	export KUBECONFIG=/tmp/kubeconfig/config
fi

CLOUD_ROOT_DOMAIN_OPTION=""
if [ -n "$TESTKUBE_CLOUD_ROOT_DOMAIN" ]; then
	CLOUD_ROOT_DOMAIN_OPTION="--cloud-root-domain $TESTKUBE_CLOUD_ROOT_DOMAIN"
fi

if [ -n "$TESTKUBE_API_KEY" ] && [ -n "$TESTKUBE_ORG_ID" ] && [ -n "$TESTKUBE_ENV_ID" ]; then
        echo "Setting the context for Testkube using API key, organization, and environment IDs"
        testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID $CLOUD_ROOT_DOMAIN_OPTION
elif [ -n "$NAMESPACE" ]; then
	echo "Setting the context for Testkube to namespace $NAMESPACE"
	testkube set context --kubeconfig --namespace $NAMESPACE
else
	missing_vars=""
	[ -z "$TESTKUBE_API_KEY" ] && missing_vars="TESTKUBE_API_KEY "
	[ -z "$TESTKUBE_ORG_ID" ] && missing_vars="${missing_vars}TESTKUBE_ORG_ID "
	[ -z "$TESTKUBE_ENV_ID" ] && missing_vars="${missing_vars}TESTKUBE_ENV_ID"
	echo "Error: Missing required Testkube variables: $missing_vars"
	exit 1
fi

for cmd in "$@"; do eval "$cmd"; done
