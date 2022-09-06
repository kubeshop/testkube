# File naming convention for test CRD generation

We support following file naming convention for Postman files to support multple environment files:

## Test filename convention

<Test name>.postman_collection.json - where `Test name` should be reused for environments files.

For example, mytest.postman_collection.json

## Test environment filename convention

<Test name>.<Test env>.postman_environment.json - where `Test name` is reused from test files and 
`Test env` is pointing to particular testing environment.

For example, mytest.prod.postman_collection.json

## Test secret environment filename convention

<Test name>.<Test env>.postman_secret_environment.json - where `Test name` is reused from test files and 
`Test env` is pointing to particular testing environment.

For example, mytest.prod.postman_secret_environment.json

It's expected that each variable value in secret environment file is provided in a form of `secret-name=secret-key`
In this case it will be added to a list of Test secret variables.
