#!/bin/zsh
set -e

TESTKUBE=${TESTKUBE:-$(which kubectl-testkube)}

script_execution_id() {
	$TESTKUBE scripts executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 8
}

test_execution_id() {
	$TESTKUBE tests executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 2
}


test_scripts_delete() {
	echo "Scripts delete test"
	$TESTKUBE scripts 
	$TESTKUBE scripts delete kubeshop-site1 > /dev/null || true
	$TESTKUBE scripts delete kubeshop-site2 > /dev/null || true
	$TESTKUBE scripts delete kubeshop-site3 > /dev/null || true
	$TESTKUBE scripts delete kubeshop-site4 > /dev/null || true
	$TESTKUBE scripts delete kubeshop-site5 > /dev/null || true
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site1
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site2 
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site3 
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site4 
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site5 
	$TESTKUBE scripts list 
}

test_scripts_delete_all() {
	echo "Scripts delete all test"
	$TESTKUBE scripts 
	$TESTKUBE scripts delete-all
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete kubeshop-site1
	$TESTKUBE scripts list 
	$TESTKUBE scripts delete-all
	$TESTKUBE scripts list 
}

test_scripts_create() {
	echo "Scripts create test"
	$TESTKUBE scripts delete kubeshop-site > /dev/null || true
	$TESTKUBE scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site
	$TESTKUBE scripts delete testkube-todo-api > /dev/null || true
	$TESTKUBE scripts create --file test/e2e/TODO.postman_collection.json --name testkube-todo-api
	$TESTKUBE scripts delete testkube-todo-frontend > /dev/null || true
	$TESTKUBE scripts create --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project
	$TESTKUBE scripts delete testkube-dashboard > /dev/null || true
	$TESTKUBE scripts create --git-uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project
	$TESTKUBE scripts delete curl-test > /dev/null || true
	cat test/e2e/curl.json | $TESTKUBE scripts create --name curl-test
}

test_scripts_run() {
	$TESTKUBE scripts run kubeshop-site -f       # postman
	$TESTKUBE scripts execution $(script_execution_id kubeshop-site)
	$TESTKUBE scripts run testkube-dashboard -f  # cypress
	$TESTKUBE scripts execution $(script_execution_id testkube-dashboard) 

	# curl issue #821 - need to be without -f
	$TESTKUBE scripts run curl-test              # curl
	sleep 5
	$TESTKUBE scripts execution $(script_execution_id curl-test) 
}


test_tests_delete_all() {
	echo "Tests delete all test"
	$TESTKUBE tests delete-all
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app1
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app2
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app3
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app4
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app5

	$TESTKUBE tests delete todo-app1
	$TESTKUBE tests list

	$TESTKUBE tests delete-all
	$TESTKUBE tests list
}

test_tests_delete() {
	echo "Tests delete test"
	$TESTKUBE tests delete todo-app1 > /dev/null || true
	$TESTKUBE tests delete todo-app2 > /dev/null || true
	$TESTKUBE tests delete todo-app3 > /dev/null || true
	$TESTKUBE tests delete todo-app4 > /dev/null || true
	$TESTKUBE tests delete todo-app5 > /dev/null || true

	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app1
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app2
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app3
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app4
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app5

	$TESTKUBE tests delete todo-app1
	$TESTKUBE tests list
	$TESTKUBE tests delete todo-app2 
	$TESTKUBE tests list
	$TESTKUBE tests delete todo-app3 
	$TESTKUBE tests list
	$TESTKUBE tests delete todo-app4 
	$TESTKUBE tests list
	$TESTKUBE tests delete todo-app5 
	$TESTKUBE tests list
}

test_tests_create() {
	echo "create tests"
	$TESTKUBE tests delete todo-app > /dev/null || true
	cat test/e2e/test-example-1.json | $TESTKUBE tests create --name todo-app
	$TESTKUBE tests delete kubeshop > /dev/null || true
	cat test/e2e/test-example-2.json | $TESTKUBE tests create --name kubeshop
}

test_tests_run() {
	echo "run tests"
	$TESTKUBE tests run todo-app -f
	$TESTKUBE tests execution $(test_execution_id todo-app) 
	$TESTKUBE tests run kubeshop -f
	$TESTKUBE tests execution $(test_execution_id kubeshop) 
}

while test $# != 0
do
    case "$1" in
    --delete-all-test) 
		test_scripts_delete_all
		test_tests_delete_all
		;;
    esac
    shift
done

test_scripts_delete
test_scripts_create 
test_scripts_run

test_tests_delete
test_tests_create
test_tests_run
