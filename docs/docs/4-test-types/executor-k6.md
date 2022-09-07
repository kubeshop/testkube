---
sidebar_position: 4
sidebar_label: K6
---
# K6 Performance Tests

**Check out our [blog post](https://kubeshop.io/blog/load-testing-in-kubernetes-with-k6-and-testkube) to follow tutorial steps to harness the power of k6 load testing in Kubernetes with Testkube's CLI and API.**

[K6](https://k6.io/docs/) Grafana k6 is an open-source load testing tool that makes performance testing easy and productive for engineering teams. K6 is free, developer-centric and extensible.

Using k6, you can test the reliability and performance of your systems and catch performance regressions and problems earlier. K6 will help you to build resilient and performant applications that scale.

K6 is developed by Grafana Labs and the community.

## **Running a K6 Test**

K6 is integral part of Testkube. The Testkube k6 executor is installed by default during the Testkube installation. To run a k6 test in Testkube you need to create a Test. 

### **Using Files as Input**

Let's save our k6 test in file e.g. `test.js`. 

```js 
import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
  http.get('https://kubeshop.github.io/testkube/');
  sleep(1);
}
```

Testkube and the k6 executor accept a test file as an input.

```bash
kubectl testkube create test --file test.js --name k6-test
```
You don't need to pass a type here, Testkube will autodetect it. 


To run the test, pass previously created test name: 

```bash 
kubectl testkube run test -f k6-test
```

You can also create a Test based on Git repository:

```bash
# create k6-test-script.js from this Git repository
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-k6.git --git-branch main --git-path examples --type "k6/script" --name k6-test-script-git
```

Testkube will clone the repository and create a Testkube Test Custom Resource in your cluster. 

### **Using Additional K6 Arguments in Your Tests**

You can also pass additional arguments to `k6` binary thanks to `--args` flag:

```bash
$ kubectl testkube run test -f k6-test --args '--vus 100 --no-connection-reuse'
```

### **K6 Test Results**

A k6 test will be successful in Testkube when all checks and thresholds are successful. In the case of an error, the test will have `failed` status, even if there is no failure in the summary report in the test logs. For details check [this k6 issue](https://github.com/grafana/k6/issues/1680).


