# Getting Started

Please follow the [install steps](/docs/installing.md) for the first installation of Testkube.

## **Getting Help**

```sh
kubectl testkube --help 

# or for scripts runs
kubectl testkube scripts --help 
```

## **Defining Tests**

After installing, you will need to add Test Scripts to your cluster. Scripts are created as a Custom Resource in Kubernetes
(access to Kubernetes cluster is required).

For now, Testkube only supports  *Postman collections*, *basic CURL execution* and *Cypress*. Stay tuned for the support of more testing tools soon.

If you don't want to create Custom Resources "by hand", we have a little helper for this: 

### **Creating Postman Collections Based Tests**

First, let's create a Postman collection:

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
kubectl testkube scripts create --file my_postman_collection.json --type "postman/collection" --name my-test-name 
```

**Note**: This is just an example of how it works. For further details you can visit the [Postman documentation](executor-postman.md).

### **Creating Cypress Tests**

Testkube supports Git based paths for Cypress projects.

You can create a new test with kubectl testkube plugin:

```sh
 kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-executor-cypress.git --git-branch jacek/feature/git-checkout --git-path examples --name test-name --type cypress/project
```

Where:

- `uri` is the git uri where Testkube will get the cypress project.
- `git-branch` is which branch should be checked out. By default the main branch will be used.
- `git-path` denotes which path of the repository should be checked out. Testkube is doing a partial git checkout so it should be fast, even for very big monorepos.
- `name` is the unique Script Custom Resource name.
- `type` is cypress/project for a Cypress based project test structure.

For now we're supporting only Cypress test runs, but plan to fully integrate it.

## **Starting a New Script Execution**

When our script is defined as a Custom Resource, we can now run it:

```shell
kubectl testkube scripts start my-test-name 

... some script run data ...

Use thefollowing command to get script execution details:
kubectl testkube scripts execution 611b6da38cd74034e7c9d408

Or watch for completition with:
kubectl testkube scripts watch 611b6da38cd74034e7c9d408

```

## **Getting Execution Details**

After the execution is complete, you can check the script details by running the
scripts execution command:

```sh
kubectl testkube scripts execution 6103a45b7e18c4ea04883866

....
some execution details
```

## **Getting Available Scripts**

To run a script execution, you'll need to know script name:

```shell
kubectl testkube scripts list

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

```shell
kubectl testkube scripts executions SCRIPT_NAME

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

## **Changing the Output Format**

For lists and details, you can use different output formats via the `--output` flag. For now we're supporting following formats:

- RAW - Raw output from the given executor (e.g., for Postman collection, it's terminal text with colors and tables).
- JSON - Test run data are encoded in JSON.
- GO - For go-template formatting (like in Docker and Kubernetes), you'll need to add the `--go-template` flag with a custom format. The default is **{{ . | printf("%+v") }}**. This will help you check available fields.


(Keep in mind that we have plans for other output formats like junit etc.)

## **Deleting a Script**

The command to delete a script is `kubectl testkube scripts delete SCRIPT_NAME`. The `--all` flag can be used to delete all.