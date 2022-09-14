#!/bin/bash

# Curl
# create the first one separately to download image if needed
testkube create test -f ../curl/curl.json --name curl-0
testkube run test curl-0
sleep 30

for i in {1..200}
do
   testkube create test -f ../curl/curl.json --name curl-$i
   for j in {1..10}
   do
      testkube run test curl-$i
   done
   sleep 10
done

# Cypress
# create the first one separately to download image if needed
tk create test --type cypress/project --git-uri https://github.com/kubeshop/testkube --git-branch cypress-tests --git-path test/cypress/executors/cypress-10 --name cypress-executor-test-0
testkube run test cypress-executor-test-0
sleep 30

for i in {1..5}
do
   tk create test --type cypress/project --git-uri https://github.com/kubeshop/testkube --git-branch cypress-tests --git-path test/cypress/executors/cypress-10 --name cypress-executor-test-$i
   for j in {1..2}
   do
      testkube run test cypress-executor-test-$i
      sleep 10
   done
   sleep 20
done
