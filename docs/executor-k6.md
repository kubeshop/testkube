# K6 Performance Tests

[K6](https://k6.io/docs/) Grafana k6 is an open-source load testing tool that makes performance testing easy and productive for engineering teams. k6 is free, developer-centric, and extensible.

Using k6, you can test the reliability and performance of your systems and catch performance regressions and problems earlier. K6 will help you to build resilient and performant applications that scale.

k6 is developed by Grafana Labs and the community.

## Running a K6 test

K6 is integral part of Testkube. Testkube K6 executor is installed by default in recent Testkube installation. To run K6 test in Testkube you need to create Test. 

### Using files as input

Let's save our K6 test in file e.g. `test.js`

```js 
import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
  http.get('https://kubeshop.github.io/testkube/');
  sleep(1);
}
```

Testkube and the K6 executor accepts a test file as an input.

```sh
kubectl testkube create test --file test.js --type "k6/script" --name k6-test
```

To run test just pass previously created test name: 

```sh 
kubectl testkube run test -f k6-test
```

You can also create Test based on Git repository:

```sh
# create k6-test-script.js from this Git repository
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-k6.git --git-branch main --git-path examples --type "k6/script" --name k6-test-script-git
```

Testkube will clone repository and create Testkube Test Custom Resource in your cluster. 

### Using additional k6 arguments in your tests

You can also pass additional arguments to `k6` binary thanks to `--args` flag:

```sh
$ kubectl testkube run test -f k6-test --args '--vus 100 --no-connection-reuse'
```

