#!/bin/zsh

set -e

script_execution_id() {
	kubectl testkube scripts executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 8
}

test_execution_id() {
	kubectl testkube tests executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 2
}


test_scripts_delete() {
	echo "Scripts delete test"
	kubectl testkube scripts 
	kubectl testkube scripts delete kubeshop-site1 > /dev/null || true
	kubectl testkube scripts delete kubeshop-site2 > /dev/null || true
	kubectl testkube scripts delete kubeshop-site3 > /dev/null || true
	kubectl testkube scripts delete kubeshop-site4 > /dev/null || true
	kubectl testkube scripts delete kubeshop-site5 > /dev/null || true
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site1
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site2 
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site3 
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site4 
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site5 
	kubectl testkube scripts list 
}

test_scripts_delete_all() {
	echo "Scripts delete all test"
	kubectl testkube scripts 
	kubectl testkube scripts delete-all
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site1
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site2
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site3
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site4
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site5
	kubectl testkube scripts list 
	kubectl testkube scripts delete kubeshop-site1
	kubectl testkube scripts list 
	kubectl testkube scripts delete-all
	kubectl testkube scripts list 
}

test_scripts_create() {
	echo "Scripts create test"
	kubectl testkube scripts delete kubeshop-site > /dev/null || true
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site
	kubectl testkube scripts delete testkube-todo-api > /dev/null || true
	kubectl testkube scripts create --file test/e2e/TODO.postman_collection.json --name testkube-todo-api
	kubectl testkube scripts delete testkube-todo-frontend > /dev/null || true
	kubectl testkube scripts create --git-branch main --uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project
	kubectl testkube scripts delete testkube-dashboard > /dev/null || true
	kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project
	kubectl testkube scripts delete curl-test > /dev/null || true
	cat test/e2e/curl.json | kubectl testkube scripts create --name curl-test
}

test_scripts_run() {
	kubectl testkube scripts run kubeshop-site -f       # postman
	kubectl testkube scripts execution $(script_execution_id kubeshop-site)
	kubectl testkube scripts run testkube-dashboard -f  # cypress
	kubectl testkube scripts execution $(script_execution_id testkube-dashboard) 

	# curl issue #821 - need to be without -f
	kubectl testkube scripts run curl-test              # curl
	sleep 5
	kubectl testkube scripts execution $(script_execution_id curl-test) 
}


test_tests_delete_all() {
	echo "Tests delete all test"
	kubectl testkube tests delete-all
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app1
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app2
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app3
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app4
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app5

	kubectl testkube tests delete todo-app1
	kubectl testkube tests list

	kubectl testkube tests delete-all
	kubectl testkube tests list
}

test_tests_delete() {
	echo "Tests delete test"
	kubectl testkube tests delete todo-app1 > /dev/null || true
	kubectl testkube tests delete todo-app2 > /dev/null || true
	kubectl testkube tests delete todo-app3 > /dev/null || true
	kubectl testkube tests delete todo-app4 > /dev/null || true
	kubectl testkube tests delete todo-app5 > /dev/null || true

	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app1
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app2
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app3
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app4
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app5

	kubectl testkube tests delete todo-app1
	kubectl testkube tests list
	kubectl testkube tests delete todo-app2 
	kubectl testkube tests list
	kubectl testkube tests delete todo-app3 
	kubectl testkube tests list
	kubectl testkube tests delete todo-app4 
	kubectl testkube tests list
	kubectl testkube tests delete todo-app5 
	kubectl testkube tests list
}

test_tests_create() {
	echo "create tests"
	kubectl testkube tests delete todo-app > /dev/null || true
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name todo-app
	kubectl testkube tests delete kubeshop > /dev/null || true
	cat test/e2e/test-example-2.json | kubectl testkube tests create --name kubeshop
}

test_tests_run() {
	echo "run tests"
	kubectl testkube tests run todo-app -f
	kubectl testkube tests execution $(test_execution_id todo-app) 
	kubectl testkube tests run kubeshop -f
	kubectl testkube tests execution $(test_execution_id kubeshop) 
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
