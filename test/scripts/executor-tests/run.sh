#!/bin/bash
set -e

# params
help='false'
delete='false'
create='false'
run='false'
schedule='false'
executor_type='all'

while getopts 'hdcrse:' flag; do
  case "${flag}" in
    h) help='true' ;; # TODO: describe params
    d) delete='true' ;;
    c) create='true' ;;
    r) run='true' ;;
    s) schedule='true' ;;
    e) executor_type="${OPTARG}" ;;
  esac
done

print_title() {
  border="=================="
  printf "$border\n  $1\n$border\n"
}

create_update_testsuite() { # testsuite_name testsuite_path
  exit_code=0
  type=""
  kubectl testkube get testsuite $1 > /dev/null 2>&1 || exit_code=$?

  if [ $exit_code == 0 ] ; then # testsuite already created
    type="update"
  else
    type="create"
  fi

  if [ "$schedule" = true ] ; then # workaround for appending schedule
    random_minute="$(($RANDOM % 59))"
    cat $2 | kubectl testkube $type testsuite --name $1 --label app=testkube --schedule "$random_minute */4 * * *" 
  else
    cat $2 | kubectl testkube $type testsuite --name $1 --label app=testkube 
  fi
}

artillery() {
  print_title "Artillery"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/artillery/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-artillery-smoke-tests -ntestkube --ignore-not-found=true
  fi

  if [ "$create" = true ] ; then
    # Tests
    kubectl apply -f test/artillery/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-artillery-smoke-tests" "test/suites/executor-artillery-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-artillery-smoke-tests
  fi
}

container() {
  print_title "Container executor"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/container-executor-curl.yaml --ignore-not-found=true
    kubectl delete -f test/container-executor/crd/curl.yaml --ignore-not-found=true
    kubectl delete testsuite executor-container-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  if [ "$create" = true ] ; then
    # Executors (not created by default)
    kubectl apply -f test/executors/container-executor-curl.yaml

    # Tests
    kubectl apply -f test/container-executor/crd/curl.yaml

    # TestsSuites
    create_update_testsuite "executor-container-smoke-tests" "test/suites/executor-container-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-container-smoke-tests
  fi
}

cypress() {
  print_title "Cypress"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml --ignore-not-found=true
    kubectl delete -f test/cypress/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-cypress-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  if [ "$create" = true ] ; then
    # Executors (not created by default)
    kubectl apply -f test/executors/cypress-v10.yaml -f test/executors/cypress-v9.yaml -f test/executors/cypress-v8.yaml

    # Tests
    kubectl apply -f test/cypress/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-cypress-smoke-tests" "test/suites/executor-cypress-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-cypress-smoke-tests
  fi
}

gradle() {
  print_title "Gradle"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/gradle-jdk-18.yaml -f test/executors/gradle-jdk-17.yaml -f test/executors/gradle-jdk-11.yaml -f test/executors/gradle-jdk-8.yaml --ignore-not-found=true
    kubectl delete -f test/gradle/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-gradle-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  if [ "$create" = true ] ; then
    # Executors (not created by default)
    kubectl apply -f test/executors/gradle-jdk-18.yaml -f test/executors/gradle-jdk-17.yaml -f test/executors/gradle-jdk-11.yaml -f test/executors/gradle-jdk-8.yaml

    # Tests
    kubectl apply -f test/gradle/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-gradle-smoke-tests" "test/suites/executor-gradle-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-gradle-smoke-tests
  fi
}

k6() {
  print_title "k6"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/k6/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-k6-smoke-tests -ntestkube --ignore-not-found=true
  fi

  if [ "$create" = true ] ; then
    # Tests
    kubectl apply -f test/k6/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-k6-smoke-tests" "test/suites/executor-k6-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-k6-smoke-tests
  fi
}

kubepug() {
  print_title "kubepug"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/kubepug/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-kubepug-smoke-tests -ntestkube --ignore-not-found=true
  fi

  if [ "$create" = true ] ; then
    # Tests
    kubectl apply -f test/kubepug/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-kubepug-smoke-tests" "test/suites/executor-kubepug-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-kubepug-smoke-tests
  fi
}

maven() {
  print_title "Maven"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/executors/maven-jdk-18.yaml -f test/executors/maven-jdk-11.yaml -f test/executors/maven-jdk-8.yaml --ignore-not-found=true
    kubectl delete -f test/maven/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-maven-smoke-tests -ntestkube --ignore-not-found=true
  fi
  
  if [ "$create" = true ] ; then
  # Executors (not created by default)
  kubectl apply -f test/executors/maven-jdk-18.yaml -f test/executors/maven-jdk-11.yaml -f test/executors/maven-jdk-8.yaml

  # Tests
  kubectl apply -f test/maven/executor-smoke/crd/crd.yaml

  # TestsSuites
  create_update_testsuite "executor-maven-smoke-tests" "test/suites/executor-maven-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-maven-smoke-tests
  fi
}

soapui() {
  print_title "soapui"
  if [ "$delete" = true ] ; then
    kubectl delete -f test/soapui/executor-smoke/crd/crd.yaml --ignore-not-found=true
    kubectl delete testsuite executor-soapui-smoke-tests -ntestkube --ignore-not-found=true
  fi

  if [ "$create" = true ] ; then
    # Tests
    kubectl apply -f test/soapui/executor-smoke/crd/crd.yaml

    # TestsSuites
    create_update_testsuite "executor-soapui-smoke-tests" "test/suites/executor-soapui-smoke-tests.json"
  fi

  if [ "$run" = true ] ; then
    testkube run testsuite executor-soapui-smoke-tests
  fi
}


main() {
  case $executor_type in
    all)
      artillery
      container
      cypress
      gradle
      k6
      kubepug
      maven
      soapui
      ;;
    artillery | container | cypress | gradle | k6 | kubepug | maven | soapui)
        $executor_type
      ;;
    *)
      echo "Error: Incorrect executor name \"$executor_type\""; exit 1
      ;;
  esac
}

main