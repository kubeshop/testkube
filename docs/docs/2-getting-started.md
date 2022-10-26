# Getting Started

Please follow the [install steps](1-installing.md) for the first installation of Testkube.

## **Getting Help**

```bash
kubectl testkube --help 

# or any other command
kubectl testkube get --help 
```

## **Launching Testkube UI**
```bash
kubectl testkube dashboard
```

## **Defining Tests**

After installing, you will need to add Tests to your cluster, which are created as a Custom Resource in Kubernetes
(access to Kubernetes cluster is required).

This document provides examples for using  *Postman collections* and *Cypress* executors. Please visit the **Test Types** section of the documentation to read about additional types of implementations.

If you don't want to create Custom Resources "by hand", we have a little helper for this: 

### **Creating Postman Collections Based Tests**

First, let's export a Postman collection from Postman UI (the file content should look similar to the one below):

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
							"    pm.expect(pm.response.text()).to.include(\"Accelerator\");",
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

```bash
kubectl testkube create test --file my_postman_collection.json --type postman/collection --name my-test-name 
```

**Note**: This is just an example of how it works. For further details you can visit the [Postman documentation](4-test-types/executor-postman.md).

### **Creating Cypress Tests**

Cypress is in the form of projects. To run them we need to pass the whole directory structure with the npm based dependencies. You can create a new test with Testkube:

```bash
 kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch main --git-path examples --name my-test-name --type cypress/project
```

Where:

- `git-uri` is the git uri where testkube will get the Cypress project.
- `git-branch` is which branch should be checked out (by default - the main branch will be used).
- `git-path` is which path of the repository should be checked out (testkube is doing a partial git checkout so it will be fast even for very big monorepos).
- `name` - is the unique Script Custom Resource name.
- `type` - cypress/project - for Cypress based project test structure.

## **Starting a New Test Execution**

When our test is defined as a Custom Resource we can now run it:

```bash
kubectl testkube run test my-test-name 

#  ... some test run output ...

```
Use the following command to get test execution details:
```bash
kubectl testkube get execution 611b6da38cd74034e7c9d408

```
Or watch for completion with
```bash
kubectl testkube watch execution 611b6da38cd74034e7c9d408

```

## **Getting Execution Details**

After the test execution is complete, access the test details by running the
tests execution command:

```bash
kubectl testkube get execution 6103a45b7e18c4ea04883866

#....some execution details....

```

## **Getting Available Scripts**

To run a test execution, you'll need to know the test name:

```bash
kubectl testkube get tests

```
Output:
```
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

## **Getting Available Executions**

```bash
kubectl testkube get executions TEST_NAME
```
Output:
```
+------------+--------------------+--------------------------+---------------------------+----------+
|   TEST     |        TYPE        |       EXECUTION ID       |      EXECUTION NAME       | STATUS   |
+------------+--------------------+--------------------------+---------------------------+----------+
| parms-test | postman/collection | 611a5a1a910ca385751eb2c6 | pt1                       | success  |
| parms-test | postman/collection | 611a5a40910ca385751eb2c8 | pt2                       | error    |
| parms-test | postman/collection | 611a5b6d910ca385751eb2ca | forcibly-frank-panda      | error    |
| parms-test | postman/collection | 611a5b83910ca385751eb2cc | slightly-merry-jennet     | error    |
| parms-test | postman/collection | 611b6d6c8cd74034e7c9d406 | frequently-expert-terrier | error    |
| parms-test | postman/collection | 611b6da38cd74034e7c9d408 | violently-fresh-elephant  | error    |
+------------+--------------------+--------------------------+---------------------------+----------+
```

## **Changing the Output Format**

For lists and details, you can use different output formats via the `--output` flag. The following formats are currently supported:

- RAW - Raw output from the given executor (e.g., for Postman collection, it's terminal text with colors and tables).
- JSON - Test run data are encoded in JSON.
- GO - For go-template formatting (like in Docker and Kubernetes), you'll need to add the `--go-template` flag with a custom format. The default is **{{ . | printf("%+v") }}**. This will help you check available fields.

There plans to support other output formats like junit etc. If there is something specific that you need, please reach out to us.

## **Deleting a Test**

The command to delete a test is `kubectl testkube delete tests TEST_NAME`. The `--all` flag can be used to delete all.
