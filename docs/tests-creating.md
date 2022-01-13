# Tests

If running single test script is not enough for you, probably you need testkube `Test` resource. 
Tests are for orchestration. Orchestration of different test steps like e.g. script execution. 

Let's assume quite big IT department where there is frontend team and backend team, everything is 
deployed on Kubernetes cluster, and each team is responsible for their part of work. Frontend enginers test their code with use of Cypress testing framework, but backend engineers prefers simplier tools like Postman, they have a lot of Postman collections defined.

There is also some QA leader who is responsible for release trains and want to be sure that before release all tests are green. 
The one issue is that he need to create pipelines which orchestrate all teams tests into some common platform. 

... it would be easy if all of them have used TestKube. 

# Tests creation

creating tests is really simple - you need to write down test definition in json file and then pass it to `testkube` `kubectl` plugin.

example test file could look like this: 

```json
cat '
{
        "name": "test-example-2",
        "namespace": "testkube",
        "description": "Example simple test orchestration",
        "steps": [
                {"type": "executeScript", "namespace": "testkube", "name": "test1"},
                {"type": "delay", "duration": 5000},
                {"type": "executeScript", "namespace": "testkube", "name": "test1"}
        ]
}' | kubectl testkube tests create
```


