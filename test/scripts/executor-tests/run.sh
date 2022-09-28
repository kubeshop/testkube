#!/bin/bash
set -e

#params
delete='false'
run='false'
executor_type=''

while getopts 'dre:' flag; do
  case "${flag}" in
    d) delete='true' ;;
    r) run='true' ;;
    e) executor_type="${OPTARG}" ;; # TODO: executor selection
  esac
done

cypress() {
    if [ "$delete" = true ] ; then
        kubectl delete -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml --ignore-not-found=true
        kubectl delete -f test/cypress/executor-smoke/crd/crd.yaml --ignore-not-found=true
        kubectl delete testsuite executors-smoke-tests -ntestkube --ignore-not-found=true
    fi
    
    # Executors (not created by default)
    kubectl apply -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml

    # Tests
    kubectl apply -f test/cypress/executor-smoke/crd/crd.yaml


    # TestsSuites
    cat test/suites/executors-smoke-tests.json | kubectl testkube create testsuite --name executors-smoke-tests --label app=testkube


}


cypress