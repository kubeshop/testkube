#!/bin/sh

if [ -n "$KUBECONFIG_FILE" ]; then
	echo "copying KUBECONFIG_FILE to /tmp/kubeconfig/config and exporting KUBECONFIG"
	echo $KUBECONFIG_FILE > /tmp/kubeconfig/config
	export KUBECONFIG=/tmp/kubeconfig/config
fi

if [ -n "$NAMESPACE" ]; then
	echo "setting the context for testkube to namespace $NAMESPACE"
	testkube set context --kubeconfig --namespace $NAMESPACE
elif [ -n "$TESTKUBE_API_KEY" ] && [ -n "$TESTKUBE_ORG_ID" ] && [ -n "TESTKUBE_ENV_ID" ]; then
	echo "setting the context for testkube pro"
	testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID
fi


for cmd in "$@"; do eval "$cmd"; done
