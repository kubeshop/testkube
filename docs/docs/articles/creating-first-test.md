# Creating Your First Test

## Kubernetes-native Tests

Tests in Testkube are stored as a Custom Resource in Kubernetes and live inside your cluster.

You can create your tests directly in the UI, using the CLI or deploy them as a Custom Resource.
Upload your test files to Testkube or provide your Git credentials so that Testkube can fetch them automatically from your Git Repo every time there's a new test execution.

This section provides an example of creating a _K6_ test. Testkube supports a long [list of testing tools](../category/test-types).

## Creating a K6 Test
Now that you have your Testkube Environment up and running, the quickest way to add a new test is by clicking "Add New Test" on the Dashboard and select your test type:
<img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/683eae92-ef74-49c8-9db9-90da76fc17fc" />

We created the following Test example which verifies the status code of an HTTPS endpoint.
```js
// This k6 test was made to fail randomly 50% of the times.
import http from 'k6/http';
import { check, fail, sleep } from 'k6';


export const options = {
 stages: [
   { duration: '1s', target: 1 },
 ],
};

let statusCode = Math.random() > 0.5 ? 200 : 502;
export default function () {
 const res = http.get('https://httpbin.test.k6.io/');
 check(res, { 'Check if status code is 200': (r) => { 
    console.log(statusCode, "Passing? ", 200 == statusCode);
    return r.status == statusCode }
});
}
```

Testkube can import any test files from Git, from your computer or by copy and pasting a string.
While in an automated setup, our advice is to keep everything in Git (including your Test CRDs).
For this example, we will copy and paste the test file to quickly create and run it.
<img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/cfb5d188-aaf6-4051-a44c-3859a23dd2a7" />



Voila! You can now run the test!
<img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/e2d46e4f-641b-49b9-8a1f-f3b3100c4ad0" />


## Different Mechanisms to Run Tests
### Dashboard
Trigger test execution manually on the Testkube Pro Dashboard:
<img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/97fe3119-60a8-4b40-ac54-3f1fc625111f" />


### CLI
You can run tests manually from your machine using the CLI as well, or from your CI/CD. Visit [here](https://docs.testkube.io/articles/cicd-overview) for examples on how to setup our CI/CD system to trigger your tests.
<img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/6b5098d7-9b57-485d-8c5e-5f915f49d515" />  

#### Changing the Output Format

For lists and details, you can use different output formats via the `--output` flag. The following formats are currently supported:

- `RAW` - Raw output from the given executor (e.g., for Postman collection, it's terminal text with colors and tables).
- `JSON` - Test run data are encoded in JSON.
- `GO` - For go-template formatting (like in Docker and Kubernetes), you'll need to add the `--go-template` flag with a custom format. The default is `{{ . | printf("%+v") }}`. This will help you check available fields.

### Other Means of Triggering Tests
- Your Test can run on a [Schedule](https://docs.testkube.io/articles/scheduling-tests)
  <img width="1896" alt="image" src="https://github.com/kubeshop/testkube/assets/13501228/aa3a1d87-e687-4364-9a8f-8bc8ffc73395" />
- Testkube can trigger the tests based on [Kubernetes events](https://docs.testkube.io/articles/test-triggers) (such as the deployment of an application).