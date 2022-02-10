# Getting Started

Please follow [install steps](/docs/installing.md) first for Testkube installation (if not installed already)

## Getting help

```sh
kubectl testkube --help 

# or for scripts runs
kubectl testkube tests --help 
```

## Defining tests

After installing you will need to add Test Scripts to your cluster, scripts are created as Custom Resource in Kubernetes
(access to Kubernetes cluster would be also needed)

For now Testkube only supports  *Postman collections*, *basic CURL execition* and *Cypress* - but we plan to handle more testing tools soon,.

If you don't want to create Custom Resources "by hand" we have a little helper for this: 

### Creating Postman Collections based tests

Fist let's create Postman collection:

```bash
cat <<EOF > my_postman_collection.json
{
	"info": {
		"_postman_id": "8af42c21-3e31-49c1-8b27-d6e60623a180",
		"name": "Kubeshop",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	},
	"item": [
		{
			"name": "Home",
			"event": [
				{
					"listen": "test",
					"script": {
						"exec": [
							"pm.test(\"Body matches string\", function () {",
							"    pm.expect(pm.response.text()).to.include(\"K8s Accelerator\");",
							"});"
						],
						"type": "text/javascript"
					}
				}
			],
			"request": {
				"method": "GET",
				"header": [],
				"url": {
					"raw": "https://kubeshop.io/",
					"protocol": "https",
					"host": [
						"kubeshop",
						"io"
					],
					"path": [
						""
					]
				}
			},
			"response": []
		},
		{
			"name": "Team",
			"event": [
				{
					"listen": "test",
					"script": {
						"exec": [
							"pm.test(\"Status code is 200\", function () {",
							"    pm.response.to.have.status(200);",
							"});"
						],
						"type": "text/javascript"
					}
				}
			],
			"request": {
				"method": "GET",
				"header": [],
				"url": {
					"raw": "https://kubeshop.io/our-team",
					"protocol": "https",
					"host": [
						"kubeshop",
						"io"
					],
					"path": [
						"our-team"
					]
				}
			},
			"response": []
		}
	]
}

EOF
```

```sh
kubectl testkube tests create --file my_postman_collection.json --type "postman/collection" --name my-test-name 
```

**Note**: this is just an example of how it works. For further details you can visit [Postman documentation](executor-postman.md)

### Creating Cypress tests

Cypress tests are little more complicated to pass - for now we're supporting Git based paths for Cypress projects.

You can create new test with kubectl testkube plugin:

```sh
 kubectl testkube tests create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch jacek/feature/git-checkout --git-path examples --name test-name --type cypress/project
```

Where:

- `uri` is git uri where testkube will get cypress project
- `git-branch` is what branch should he checkout (default - main branch will be used)
- `git-path` is what path of repository should be checked out (testkube is doing partial git checkout so it'll be fast even for very big monorepos)
- `name` - is unique Sript Custom Resource name.
- `type` - cypress/project - for Cypress based project test structure

For now we're supporting only Cypress test runs, but we plan to fully integrate it.

## Starting new test execution

When our test is defined as CR we can now run it:

```shell
kubectl testkube tests start my-test-name 

... some test run data ...

Use following command to get test execution details:
kubectl testkube tests execution 611b6da38cd74034e7c9d408

or watch for completition with
kubectl testkube tests watch 611b6da38cd74034e7c9d408

```

## Getting execution details

After test completed with success or error you can go back to test details by running
scripts execution command:

```sh
kubectl testkube tests execution 6103a45b7e18c4ea04883866

....
some execution details
```

## Getting available scripts

To run test execution you'll need to know test name

```shell
kubectl testkube tests list

+----------------------+--------------------+
|         NAME         |        TYPE        |
+----------------------+--------------------+
| create-001-test      | postman/collection |
| envs1                | postman/collection |
| test-kubeshop        | postman/collection |
| test-kubeshop-failed | postman/collection |
| test-postman-script  | postman/collection |
+----------------------+--------------------+

```

## Getting available executions

```shell
kubectl testkube tests executions SCRIPT_NAME

+------------+--------------------+--------------------------+---------------------------+----------+
|   SCRIPT   |        TYPE        |       EXECUTION ID       |      EXECUTION NAME       | STATUS   |
+------------+--------------------+--------------------------+---------------------------+----------+
| parms-test | postman/collection | 611a5a1a910ca385751eb2c6 | pt1                       | success  |
| parms-test | postman/collection | 611a5a40910ca385751eb2c8 | pt2                       | error    |
| parms-test | postman/collection | 611a5b6d910ca385751eb2ca | forcibly-frank-panda      | error    |
| parms-test | postman/collection | 611a5b83910ca385751eb2cc | slightly-merry-jennet     | error    |
| parms-test | postman/collection | 611b6d6c8cd74034e7c9d406 | frequently-expert-terrier | error    |
| parms-test | postman/collection | 611b6da38cd74034e7c9d408 | violently-fresh-elephant  | error    |
+------------+--------------------+--------------------------+---------------------------+----------+
```

## Changing output format

For lists and details you can use different output format (`--output` flag) for now we're supporting following formats:

- RAW - it's raw output from given executor (e.g. for Postman collection it's terminal text with colors and tables)
- JSON - test run data are encoded in JSON
- GO - go-template formatting (like in Docker and Kubernetes) you'll need to add `--go-template` flag with custom format (defaults to {{ . | printf("%+v") }} will help you check available fields)

(Keep in mind that we have plans for other output formats like junit etc.)

## Deleting a test

For deleting a test there is `kubectl testkube tests delete SCRIPT_NAME` command and also `--all` flag can be used to delete all.
