#!/usr/bin/env sh

# -------- testkube test suite tests ----------

kubectl delete test testkube-dashboard -ntestkube || true
kubectl delete secret testkube-dashboard-secrets -ntestkube || true
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project

kubectl delete test testkube-api -ntestkube || true
kubectl delete secret testkube-api-secrets -ntestkube || true
kubectl testkube create test -f test/postman/Testkube-API.postman_collection.json --name testkube-api

kubectl delete test testkube-api-failing -ntestkube || true
kubectl delete secret testkube-api-failing-secrets -ntestkube || true
kubectl testkube create test -f test/postman/Testkube-API-Failing.postman_collection.json --name testkube-api-failing

kubectl delete test testkube-homepage-performance -ntestkube || true
kubectl delete secret testkube-homepage-performance-secrets -ntestkube || true
kubectl testkube create test --file test/perf/testkube-homepage.js --type "k6/script" --name testkube-homepage-performance

kubectl delete test testkube-api-performance -ntestkube || true
kubectl delete secret testkube-api-performance-secrets -ntestkube || true
kubectl testkube create test --file test/perf/api-server.js --type "k6/script" --name testkube-api-performance


# -------- other tests ----------

kubectl delete test testkube-todo-frontend -ntestkube || true
kubectl delete secret testkube-todo-frontend-secrets -ntestkube || true
kubectl testkube create test --git-branch main --git-uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project

kubectl delete test testkube-todo-api -ntestkube || true
kubectl delete secret testkube-todo-api-secrets -ntestkube || true
kubectl testkube create test --file test/postman/TODO.postman_collection.json --name testkube-todo-api

kubectl delete test kubeshop-site -ntestkube || true
kubectl delete secret kubeshop-site-secrets -ntestkube || true
kubectl testkube create test --file test/postman/Kubeshop.postman_collection.json --name kubeshop-site 

# --------- test suites definitions ---------

kubectl delete testsuite testkube -ntestkube || true
cat test/suites/testsuite-testkube.json | kubectl testkube create testsuite --name testkube --label app=testkube

kubectl delete testsuite testkube-failing -ntestkube || true
cat test/suites/testsuite-testkube-failing.json | kubectl testkube create testsuite --name testkube-failing --label app=testkube

kubectl delete testsuite testkube-failing-stop -ntestkube || true
cat test/suites/testsuite-testkube-failing-sof.json | kubectl testkube create testsuite --name testkube-failing-stop --label app=testkube


kubectl delete testsuite testkube-global-test -ntestkube || true
cat test/suites/testsuite-example-1.json | kubectl testkube create testsuite --name testkube-global-test --label app=mixed

kubectl delete testsuite kubeshop-sites-test -ntestkube || true
cat test/suites/testsuite-example-2.json | kubectl testkube create testsuite --name kubeshop-sites-test  --label app=sites