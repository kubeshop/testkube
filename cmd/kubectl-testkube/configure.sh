#!/bin/sh

if [ -n "$KUBECONFIG_FILE" ]; then
	echo "copying KUBECONFIG_FILE to /tmp/kubeconfig/config and exporting KUBECONFIG"
	$KUBECONFIG_FILE > /tmp/kubeconfig/config
	export KUBECONFIG=/tmp/kubeconfig/config
fi

if [ -n "$NAMESPACE" ]; then
	echo "setting the context for testkube to namespace $NAESPACE"
	testkube set context --kubeconfig --namespace $NAMESPACE
fi

if [ -n "$TESTKUBE_API_KEY" ] && [ -n "$TESTKUBE_ORG_ID" ] && [ -n "TESTKUBE_ENV_ID" ]; then
	echo "setting the context for testkube pro"
	testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID
fi


for cmd in "$@"; do eval "$cmd"; done
