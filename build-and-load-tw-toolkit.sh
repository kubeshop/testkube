#!/bin/bash
set -eo pipefail

AGENT=double-o-seven

# Build

tk-dev build -c=a

# Setup depot on agent namespace
if ! (kubectl get secret depot -n ${AGENT} &>/dev/null); then
  kubectl create secret --namespace ${AGENT} docker-registry depot --docker-server=registry.depot.dev --docker-username=x-token --docker-password=${DEPOT_TOKEN}
fi

BUILD_ID=$(jq -r '."depot.build".buildID' depot-build-meta.json)
echo deploying build ${BUILD_ID}

# Update runner image first
kubectl patch deployment testkube-${AGENT}-testkube-runner -n ${AGENT} -p '{"spec":{"template":{"spec":{"imagePullSecrets":[{"name":"depot"}]}}}}'
kubectl set image deployment/testkube-${AGENT}-testkube-runner -n ${AGENT} testkube-runner=registry.depot.dev/3cp8bwpbj0:${BUILD_ID}-agent-server

# Set toolkit image versions to latest build
kubectl set env deployment/testkube-${AGENT}-testkube-runner -n ${AGENT} \
  TESTKUBE_TW_TOOLKIT_IMAGE=registry.depot.dev/3cp8bwpbj0:${BUILD_ID}-testworkflow-toolkit \
  TESTKUBE_TW_INIT_IMAGE=registry.depot.dev/3cp8bwpbj0:${BUILD_ID}-testworkflow-init \
  TESTKUBE_GLOBAL_WORKFLOW_TEMPLATE_INLINE='{"pod":{"imagePullSecrets":[{"name":"depot"}]}}'

