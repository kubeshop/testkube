#!/usr/bin/env bash

# Wait for Istio's proxy to be ready, otherwise requests could fail.
# This is only applicable in environments where the holdApplicationUntilProxyStarts
# feature is unavailable.
if [[ "${ISTIO_PROXY_WAIT}" == "true" ]]; then
    echo -n "Waiting for Istio's proxy to become ready..."
    until curl -fsI http://localhost:15021/healthz/ready &> /dev/null; do echo -n "."; sleep 3; done;
    echo OK;
fi

# Execute the runner
/bin/runner "$@";
runner_exit_code=$(echo $?);

# Send signal to Istio's proxy, otherwise it will keep running preventing
# the job from completing.
# This is only applicable to envrionments where the native sidecar solution
# is not available:
# https://istio.io/latest/blog/2023/native-sidecars/
if [[ "${ISTIO_PROXY_EXIT}" == "true" ]]; then
    echo -n "Signal to Istio's proxy that it may exit..."
    until curl -fsI -X POST http://localhost:15020/quitquitquit &> /dev/null; do echo -n "."; sleep 3; done;
    echo OK;
fi

# Exit
exit $runner_exit_code
