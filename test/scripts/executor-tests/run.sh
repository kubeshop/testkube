#!/bin/bash
set -e

# params
delete='false'
run='false'
executor_type='all'
help='false'

while getopts 'hdre:' flag; do
  case "${flag}" in
    h) help='true' ;; # TODO: describe params
    d) delete='true' ;;
    r) run='true' ;;
    e) executor_type="${OPTARG}" ;; # TODO: executor selection
  esac
done

print_title() {
  border="=================="
  printf "$border\n  $1\n$border\n"
}

artillery_create() {
  print_title "Artillery - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/artillery/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-artillery-smoke-tests -ntestkube --ignore-not-found=true
  fi

  # Tests
  kubectl apply -f test/artillery/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-artillery-smoke-tests.json | kubectl testkube create testsuite --name executor-artillery-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
}

container_executor_create() {
  print_title "Container executor - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/container-executor-curl.yaml --ignore-not-found=true
    kubectl delete -f test/container-executor/crd/curl.yaml --ignore-not-found=true
    kubectl delete testsuite executor-container-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  # Executors (not created by default)
  kubectl apply -f test/executors/container-executor-curl.yaml

  # Tests
  kubectl apply -f test/container-executor/crd/curl.yaml

  # TestsSuites
  cat test/suites/executor-container-smoke-tests.json | kubectl testkube create testsuite --name executor-container-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
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

kubepug_create() {
  print_title "kubepug - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/kubepug/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-kubepug-smoke-tests -ntestkube --ignore-not-found=true
  fi

  # Tests
  kubectl apply -f test/kubepug/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-kubepug-smoke-tests.json | kubectl testkube create testsuite --name executor-kubepug-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
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

soapui_create() {
  print_title "soapui - create"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/soapui/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-soapui-smoke-tests -ntestkube --ignore-not-found=true
  fi

  # Tests
  kubectl apply -f test/soapui/executor-smoke/crd/crd.yaml

  # TestsSuites
  cat test/suites/executor-soapui-smoke-tests.json | kubectl testkube create testsuite --name executor-soapui-smoke-tests --label app=testkube # TODO: will fail if Testsuite is already created (and not removed)
}

run() {
  case $executor_type in
    all)
      artillery_create
      container_executor_create
      cypress_create
      gradle_create
      k6_create
      kubepug
      maven_create
      soapui
      ;;
    artillery)
      artillery_create
      ;;
    container)
      container_executor_create
      ;;
    cypress)
      cypress_create
      ;;
    gradle)
      gradle_create
      ;;
    k6)
      k6_create
      ;;
    kubepug)
      kubepug_create
      ;;
    maven)
      maven_create
      ;;
    soapui)
      soapui_create
      ;;
    *)
      echo "Error: Incorrect executor name \"$executor_type\""; exit 1
      ;;
  esac
}

run