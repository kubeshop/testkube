#!/bin/bash
set -e

# params
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

print_title() {
  border="=================="
  printf "$border\n  $1\n$border\n"
}

cypress_create() {
  print_title "Cypress - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml --ignore-not-found=true
    kubectl delete -f test/cypress/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-cypress-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  # Executors (not created by default)
  kubectl apply -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml

  # Tests
  kubectl apply -f test/cypress/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-cypress-smoke-tests.json | kubectl testkube create testsuite --name executor-cypress-smoke-tests --label app=testkube
}

k6_create() {
  print_title "k6 - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/k6/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-k6-smoke-tests -ntestkube --ignore-not-found=true
  fi

  # Tests
  kubectl apply -f test/k6/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-k6-smoke-tests.json | kubectl testkube create testsuite --name executor-k6-smoke-tests --label app=testkube
}

run() {
  cypress_create
  k6_create
}

run