#!/bin/zsh
set -e

TESTKUBE=${TESTKUBE:-$(which kubectl-testkube)}

test_execution_id() {
	$TESTKUBE get executions | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 2
}

testsuite_execution_id() {
	$TESTKUBE get tse | grep $1 | head -n 1 | tr -s ' ' | cut -d" " -f 2
}


test_tests_delete() {
	echo "Tests delete test"
	$TESTKUBE delete test kubeshop-site1 > /dev/null || true
	$TESTKUBE delete test kubeshop-site2 > /dev/null || true
	$TESTKUBE delete test kubeshop-site3 > /dev/null || true
	$TESTKUBE delete test kubeshop-site4 > /dev/null || true
	$TESTKUBE delete test kubeshop-site5 > /dev/null || true

	kubectl delete secrets kubeshop-site1-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site2-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site3-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site4-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site5-secrets > /dev/null || true

	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site1
	kubectl delete secrets kubeshop-site1-secrets > /dev/null || true
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site2
	kubectl delete secrets kubeshop-site2-secrets > /dev/null || true
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site3
	kubectl delete secrets kubeshop-site3-secrets > /dev/null || true
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site4
	kubectl delete secrets kubeshop-site4-secrets > /dev/null || true
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site5
	kubectl delete secrets kubeshop-site5-secrets > /dev/null || true
	$TESTKUBE get tests
}

test_tests_delete_all() {
	echo "Tests delete all test"
	$TESTKUBE tests
	$TESTKUBE delete tests --all

	# delete secrets (for now manually)
	# TODO change it after deletion of secrets will arrive to delete test
	kubectl delete secrets kubeshop-site1-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site2-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site3-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site4-secrets > /dev/null || true
	kubectl delete secrets kubeshop-site5-secrets > /dev/null || true

	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site1
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site2
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site3
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site4
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site5
	$TESTKUBE get tests
	$TESTKUBE delete test kubeshop-site1
	$TESTKUBE get tests
	$TESTKUBE delete tests --all
	$TESTKUBE get tests
}

test_tests_create() {
	echo "Tests create test"
	$TESTKUBE delete test kubeshop-site > /dev/null || true
	kubectl delete secrets kubeshop-site-secrets > /dev/null || true
	$TESTKUBE create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site
	$TESTKUBE delete test testkube-todo-api > /dev/null || true
	kubectl delete secrets kubeshop-todo-api-secrets > /dev/null || true
	$TESTKUBE create test --file test/postman/TODO.postman_collection.json --name testkube-todo-api
	$TESTKUBE delete test testkube-todo-frontend > /dev/null || true
	kubectl delete secrets kubeshop-todo-frontend-secrets > /dev/null || true
	$TESTKUBE create test --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project
	$TESTKUBE delete test testkube-dashboard > /dev/null || true
	kubectl delete secrets kubeshop-dashboard-secrets > /dev/null || true
	$TESTKUBE create test --git-uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project
	$TESTKUBE delete test curl-test > /dev/null || true
	kubectl delete secrets curl-test-secrets > /dev/null || true
	cat test/curl/curl.json | $TESTKUBE create test --name curl-test
}

test_tests_run() {
	$TESTKUBE run test kubeshop-site -f       # postman
	$TESTKUBE get execution $(test_execution_id kubeshop-site)
	$TESTKUBE run test testkube-dashboard -f  # cypress
	$TESTKUBE get execution $(test_execution_id testkube-dashboard)

	# curl issue #821 - need to be without -f
	$TESTKUBE run test curl-test              # curl
	sleep 5
	$TESTKUBE get execution $(test_execution_id curl-test)
}


test_tests_delete_all() {
	echo "Tests delete all test"
	$TESTKUBE delete testsuites --all
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app1
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app2
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app3
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app4
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app5

	$TESTKUBE delete testsuites todo-app1
	$TESTKUBE get testsuites

	$TESTKUBE delete testsuites --all
	$TESTKUBE get testsuites
}

test_testsuites_delete() {
	echo "Tests delete test"
	$TESTKUBE delete testsuites todo-app1 > /dev/null || true
	$TESTKUBE delete testsuites todo-app2 > /dev/null || true
	$TESTKUBE delete testsuites todo-app3 > /dev/null || true
	$TESTKUBE delete testsuites todo-app4 > /dev/null || true
	$TESTKUBE delete testsuites todo-app5 > /dev/null || true

	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app1
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app2
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app3
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app4
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app5

	$TESTKUBE delete testsuites todo-app1
	$TESTKUBE get testsuites
	$TESTKUBE delete testsuites todo-app2
	$TESTKUBE get testsuites
	$TESTKUBE delete testsuites todo-app3
	$TESTKUBE get testsuites
	$TESTKUBE delete testsuites todo-app4
	$TESTKUBE get testsuites
	$TESTKUBE delete testsuites todo-app5
	$TESTKUBE get testsuites
}

test_testsuites_create() {
	echo "create tests"
	$TESTKUBE delete testsuites todo-app > /dev/null || true
	cat test/suites/testsuite-example-1.json | $TESTKUBE create testsuite --name todo-app
	$TESTKUBE delete testsuites kubeshop > /dev/null || true
	cat test/suites/testsuite-example-2.json | $TESTKUBE create testsuite --name kubeshop
}

test_testsuites_run() {
	echo "run tests"
	$TESTKUBE run testsuite todo-app -f
	$TESTKUBE testsuites execution $(testsuite_execution_id todo-app)
	$TESTKUBE run testsuite kubeshop -f
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
