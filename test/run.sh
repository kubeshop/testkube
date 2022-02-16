#!/bin/zsh
set -e

TESTKUBE=${TESTKUBE:-$(which kubectl-testkube)}

test_execution_id() {
	$TESTKUBE tests executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 8
}

testsuite_execution_id() {
	$TESTKUBE testsuites executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 2
}


test_tests_delete() {
	echo "Tests delete test"
	$TESTKUBE tests 
	$TESTKUBE tests delete kubeshop-site1 > /dev/null || true
	$TESTKUBE tests delete kubeshop-site2 > /dev/null || true
	$TESTKUBE tests delete kubeshop-site3 > /dev/null || true
	$TESTKUBE tests delete kubeshop-site4 > /dev/null || true
	$TESTKUBE tests delete kubeshop-site5 > /dev/null || true
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site1
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site2 
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site3 
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site4 
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site5 
	$TESTKUBE tests list 
}

test_tests_delete_all() {
	echo "Tests delete all test"
	$TESTKUBE tests 
	$TESTKUBE tests delete-all
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE tests list 
	$TESTKUBE tests delete kubeshop-site1
	$TESTKUBE tests list 
	$TESTKUBE tests delete-all
	$TESTKUBE tests list 
}

test_tests_create() {
	echo "Tests create test"
	$TESTKUBE tests delete kubeshop-site > /dev/null || true
	$TESTKUBE tests create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site
	$TESTKUBE tests delete testkube-todo-api > /dev/null || true
	$TESTKUBE tests create --file test/e2e/TODO.postman_collection.json --name testkube-todo-api
	$TESTKUBE tests delete testkube-todo-frontend > /dev/null || true
	$TESTKUBE tests create --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project
	$TESTKUBE tests delete testkube-dashboard > /dev/null || true
	$TESTKUBE tests create --git-uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project
	$TESTKUBE tests delete curl-test > /dev/null || true
	cat test/e2e/curl.json | $TESTKUBE tests create --name curl-test
}

test_tests_run() {
	$TESTKUBE tests run kubeshop-site -f       # postman
	$TESTKUBE tests execution $(test_execution_id kubeshop-site)
	$TESTKUBE tests run testkube-dashboard -f  # cypress
	$TESTKUBE tests execution $(test_execution_id testkube-dashboard) 

	# curl issue #821 - need to be without -f
	$TESTKUBE tests run curl-test              # curl
	sleep 5
	$TESTKUBE tests execution $(test_execution_id curl-test) 
}


test_tests_delete_all() {
	echo "Tests delete all test"
	$TESTKUBE testsuites delete-all
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app1
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app2
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app3
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app4
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app5

	$TESTKUBE testsuites delete todo-app1
	$TESTKUBE testsuites list

	$TESTKUBE testsuites delete-all
	$TESTKUBE testsuites list
}

test_testsuites_delete() {
	echo "Tests delete test"
	$TESTKUBE testsuites delete todo-app1 > /dev/null || true
	$TESTKUBE testsuites delete todo-app2 > /dev/null || true
	$TESTKUBE testsuites delete todo-app3 > /dev/null || true
	$TESTKUBE testsuites delete todo-app4 > /dev/null || true
	$TESTKUBE testsuites delete todo-app5 > /dev/null || true

	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app1
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app2
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app3
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app4
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app5

	$TESTKUBE testsuites delete todo-app1
	$TESTKUBE testsuites list
	$TESTKUBE testsuites delete todo-app2 
	$TESTKUBE testsuites list
	$TESTKUBE testsuites delete todo-app3 
	$TESTKUBE testsuites list
	$TESTKUBE testsuites delete todo-app4 
	$TESTKUBE testsuites list
	$TESTKUBE testsuites delete todo-app5 
	$TESTKUBE testsuites list
}

test_testsuites_create() {
	echo "create tests"
	$TESTKUBE testsuites delete todo-app > /dev/null || true
	cat test/e2e/testsuite-example-1.json | $TESTKUBE testsuites create --name todo-app
	$TESTKUBE testsuites delete kubeshop > /dev/null || true
	cat test/e2e/testsuite-example-2.json | $TESTKUBE testsuites create --name kubeshop
}

test_testsuites_run() {
	echo "run tests"
	$TESTKUBE testsuites run todo-app -f
	$TESTKUBE testsuites execution $(testsuite_execution_id todo-app)
	$TESTKUBE testsuites run kubeshop -f
	$TESTKUBE testsuites execution $(testsuite_execution_id kubeshop)
}

while test $# != 0
do
    case "$1" in
    --delete-all-test) 
		test_tests_delete_all
		test_tests_delete_all
		;;
    esac
    shift
done

test_tests_delete
test_tests_create
test_tests_run

test_testsuites_delete
test_testsuites_create
test_testsuites_run
