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
  cat test/suites/executor-cypress-smoke-tests.json | kubectl testkube create testsuite --name executor-cypress-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
}

gradle_create() {
  print_title "Gradle - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/gradle-jdk-18.yaml -f test/executors/gradle-jdk-17.yaml -f test/executors/gradle-jdk-11.yaml -f test/executors/gradle-jdk-8.yaml --ignore-not-found=true
    kubectl delete -f test/gradle/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-gradle-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  # Executors (not created by default)
  kubectl apply -f test/executors/gradle-jdk-18.yaml -f test/executors/gradle-jdk-17.yaml -f test/executors/gradle-jdk-11.yaml -f test/executors/gradle-jdk-8.yaml

  # Tests
  kubectl apply -f test/gradle/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-gradle-smoke-tests.json | kubectl testkube create testsuite --name executor-gradle-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
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
  cat test/suites/executor-k6-smoke-tests.json | kubectl testkube create testsuite --name executor-k6-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
}

maven_create() {
  print_title "Maven - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/maven-jdk-18.yaml -f test/executors/maven-jdk-11.yaml -f test/executors/maven-jdk-8.yaml --ignore-not-found=true
    kubectl delete -f test/maven/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-maven-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  # Executors (not created by default)
  kubectl apply -f test/executors/maven-jdk-18.yaml -f test/executors/maven-jdk-11.yaml -f test/executors/maven-jdk-8.yaml

  # Tests
  kubectl apply -f test/maven/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-maven-smoke-tests.json | kubectl testkube create testsuite --name executor-maven-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
}

run() {
  cypress_create
  gradle_create
  k6_create
  maven_create
}

run