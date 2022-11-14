#!/bin/bash
set -e

# params
help='false'
delete='false'
create='false'
run='false'
follow='false'
schedule='false'
executor_type='all'
custom_testsuite=''

while getopts 'hdcrfse:t:v' flag; do
  case "${flag}" in
    h) help='true' ;; # TODO: describe params
    d) delete='true' ;;
    c) create='true' ;;
    r) run='true' ;;
    f) follow='true' ;;
    s) schedule='true' ;;
    e) executor_type="${OPTARG}" ;;
    t) custom_testsuite="${OPTARG}" ;;
    v) set -x ;;
  esac
done

print_title() {
  border="=================="
  printf "\n$border\n===  $1\n$border\n"
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

run_follow_testsuite() { # testsuite_name
  follow_param=''
  if [ "$follow" = true ] ; then
    follow_param=' -f'
  fi

  testkube run testsuite $1 $follow_param
}

common_run() { # name, test_crd_file, testsuite_name, testsuite_file, custom_executor_crd_file
  name=$1
  test_crd_file=$2
  testsuite_name=$3
  testsuite_file=$4
  custom_executor_crd_file=$5

  print_title "$name"

  if [ "$delete" = true ] ; then
    if [ ! -z "$custom_executor_crd_file" ] ; then
      kubectl delete -f $custom_executor_crd_file --ignore-not-found=true
    fi
    kubectl delete -f $test_crd_file --ignore-not-found=true
    kubectl delete testsuite $testsuite_name -ntestkube --ignore-not-found=true
  fi

  if [ "$create" = true ] ; then
    if [ ! -z "$custom_executor_crd_file" ] ; then
      # Executors (not created by default)
      kubectl apply -f $custom_executor_crd_file
    fi
    
    # Tests
    kubectl apply -f $test_crd_file

    # TestsSuites
    create_update_testsuite "$testsuite_name" "$testsuite_file"
  fi

  if [ "$run" = true ] && [ "$custom_testsuite" = '' ]; then
    run_follow_testsuite $testsuite_name
  fi
}

artillery-smoke() {
  name="artillery"
  test_crd_file="test/artillery/executor-smoke/crd/crd.yaml"
  testsuite_name="executor-artillery-smoke-tests"
  testsuite_file="test/suites/executor-artillery-smoke-tests.json"
  
  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

container-smoke() {
  name="Container executor"
  test_crd_file="test/container-executor/executor-smoke/crd/curl.yaml"
  testsuite_name="executor-container-smoke-tests"
  testsuite_file="test/suites/executor-container-smoke-tests.json"

  custom_executor_crd_file="test/executors/container-executor-curl.yaml"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file" "$custom_executor_crd_file"
}

curl-smoke() {
  name="curl"
  test_crd_file="test/curl/executor-tests/crd/smoke.yaml"
  testsuite_name="executor-curl-smoke-tests"
  testsuite_file="test/suites/executor-curl-smoke-tests.json"
  
  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

cypress-smoke() {
  name="Cypress"
  test_crd_file="test/cypress/executor-tests/crd/crd.yaml"
  testsuite_name="executor-cypress-smoke-tests"
  testsuite_file="test/suites/executor-cypress-smoke-tests.json"

  custom_executor_crd_file="test/executors/cypress.yaml"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file" "$custom_executor_crd_file"
}

gradle-smoke() {
  name="Gradle"
  test_crd_file="test/gradle/executor-smoke/crd/crd.yaml"
  testsuite_name="executor-gradle-smoke-tests"
  testsuite_file="test/suites/executor-gradle-smoke-tests.json"

  custom_executor_crd_file="test/executors/gradle.yaml"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file" "$custom_executor_crd_file"
}

jmeter-smoke() {
  name="JMeter"
  test_crd_file="test/jmeter/executor-tests/crd/smoke.yaml"
  testsuite_name="executor-jmeter-smoke-tests"
  testsuite_file="test/suites/executor-jmeter-smoke-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

k6-smoke() {
  name="k6 smoke"
  test_crd_file="test/k6/executor-tests/crd/smoke.yaml"
  testsuite_name="executor-k6-smoke-tests"
  testsuite_file="test/suites/executor-k6-smoke-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

k6-other() {
  name="k6 other"
  test_crd_file="test/k6/executor-tests/crd/other.yaml"
  testsuite_name="executor-k6-other-tests"
  testsuite_file="test/suites/executor-k6-other-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

kubepug-smoke() {
  name="kubepug"
  test_crd_file="test/kubepug/executor-smoke/crd/crd.yaml"
  testsuite_name="executor-kubepug-smoke-tests"
  testsuite_file="test/suites/executor-kubepug-smoke-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

maven-smoke() {
  name="Maven"
  test_crd_file="test/maven/executor-smoke/crd/crd.yaml"
  testsuite_name="executor-maven-smoke-tests"
  testsuite_file="test/suites/executor-maven-smoke-tests.json"

  custom_executor_crd_file="test/executors/maven.yaml"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file" "$custom_executor_crd_file"
}

postman-smoke() {
  name="postman"
  test_crd_file="test/postman/executor-tests/crd/crd.yaml"
  testsuite_name="executor-postman-smoke-tests"
  testsuite_file="test/suites/executor-postman-smoke-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

soapui-smoke() {
  name="SoapUI"
  test_crd_file="test/soapui/executor-smoke/crd/crd.yaml"
  testsuite_name="executor-soapui-smoke-tests"
  testsuite_file="test/suites/executor-soapui-smoke-tests.json"

  common_run "$name" "$test_crd_file" "$testsuite_name" "$testsuite_file"
}

main() {
  case $executor_type in
    all)
      artillery-smoke
      container-smoke
      curl-smoke
      cypress-smoke
      gradle-smoke
      jmeter-smoke
      k6-smoke
      k6-other
      kubepug-smoke
      maven-smoke
      postman-smoke
      soapui-smoke
      ;;
    smoke)
      artillery-smoke
      container-smoke
      curl-smoke
      cypress-smoke
      gradle-smoke
      jmeter-smoke
      k6-smoke
      kubepug-smoke
      maven-smoke
      postman-smoke
      soapui-smoke
      ;;
    *)
      $executor_type
      ;;
  esac

  if [ "$custom_testsuite" != '' ] ; then # create/delete/schedule all resources, but execute only ones from Custom Testsuite
    filename=$(basename $custom_testsuite)
    testsuite_name="${filename%%.*}"

    create_update_testsuite "$testsuite_name" "$custom_testsuite"
    run_follow_testsuite "$testsuite_name"
  fi
}

main