---
sidebar_position: 7
sidebar_label: Maven
---
# Maven-based Tests

Testkube allows us to run Maven-based tasks which could be also tests. For example we can easily run JUnit tests in Testkube now. 


## **Test Environment**

We'll try to put simple JUnit test to our cluster and run it. Testkube Maven Executor handles `mvn` and `mvnw` binaries.
Because Maven projects are quite complicated in terms of directory structure. We'll need to load them from a Git directory.

You can find example projects in the repository here: https://github.com/kubeshop/testkube-executor-maven/tree/main/examples.

Let's create a simple test which will check if an env variable is set to true: 
```java
package hello.maven;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class LibraryTest {
    @Test void someLibraryMethodReturnsTrue() {
        String env = System.getenv("TESTKUBE_MAVEN");
        assertTrue(Boolean.parseBoolean(env), "TESTKUBE_MAVEN env should be true");
    }
}
```


The default Maven executor: 

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: maven-executor
  namespace: testkube
spec:
  image: kubeshop/testkube-maven-executor:0.1.4
  types:
  - maven/project
  - maven/test
  - maven/integration-test 
```

As we can see, there are several types. The Maven executor handles the second part after `/` as a task name, so `maven/test` will run `mvn test` and so on. 

One exception from this rule is `project` which is a generic one and forces you to pass additional arguments during test execution. For example:

```bash
kubectl testkube run maven-example-project --args='runMyCustomTask' 
```


## **Create a New Maven-based Test**

```bash
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-maven.git --git-path examples/hello-maven --type maven/test --name maven-example-test --git-branch main
```



## **Running a Test**

Let's pass the env variable to our test run:

```bash
kubectl testkube run test maven-example-test -f -v TESTKUBE_MAVEN=true

# ...... after some time

Test execution completed with success in 16.555s 🥇

Watch the test execution until complete:
$ kubectl testkube watch execution 62d148db0260f256c1a1e993


Use the following command to get test execution details:
$ kubectl testkube get execution 62d148db0260f256c1a1e993
```

## **Getting Test Results**

Now we can watch/get test execution details:

```bash
kubectl testkube get execution 62d148db0260f256c1a1e993
```

Output:

```bash
# ....... a lot of Maven logs

Downloaded from central: https://repo.maven.apache.org/maven2/org/junit/platform/junit-platform-launcher/1.7.2/junit-platform-launcher-1.7.2.pom (3.0 kB at 121 kB/s)
[INFO] 
[INFO] -------------------------------------------------------
[INFO]  T E S T S
[INFO] -------------------------------------------------------
[INFO] Running hello.maven.LibraryTest
[INFO] Tests run: 1, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 0.052 s - in hello.maven.LibraryTest
[INFO] 
[INFO] Results:
[INFO] 
[INFO] Tests run: 1, Failures: 0, Errors: 0, Skipped: 0
[INFO] 
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time:  9.851 s
[INFO] Finished at: 2022-07-18T09:06:15Z
[INFO] ------------------------------------------------------------------------

Status Test execution completed with success 🥇
```

## Using different JDKs 

In the Java world, usually you want to have control over your Runtime environment. Testkube can easily handle that for you! 
We're building several Java images to handle constraints which Maven can put in it's build file.

To use a different executor you can use one of our pre-built ones (for Java 8,11,17,18) or build your own Docker image based on a Maven executor.

Let's assume we need JDK18 for our test runs. To handle that issue, create a new Maven executor:

content of `maven-jdk18-executor.yaml`
```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: maven-jdk18-executor
  namespace: testkube
spec:
  image: kubeshop/testkube-maven-executor:0.1.0-jdk18   # <-- we're building jdk
  types:
  - maven:jdk18/project # <-- just create different test type with naming convention "framework:version/type"
  - maven:jdk18/test
  - maven:jdk18/integration-test 
```

> Tip: Look for recent executor versions here https://hub.docker.com/repository/registry-1.docker.io/kubeshop/testkube-maven-executor/tags?page=1&ordering=last_updated.


And add it to your cluster: 
```bash
kubectl apply -f maven-jdk18-executor.yaml 
```

Now, create a new test with a type which our new executor can handle e.g.: `maven:jdk18/test`

```bash 
 # create test
 kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-maven.git --git-path examples/hello-maven-jdk18 --type maven:jdk18/test --name maven-jdk18-example-test --git-branch main

# and run it
kubectl testkube run test maven-jdk18-example-test -f -v TESTKUBE_MAVEN=true
```


## **Summary**

Testkube simplifies running Java tests based on Maven and simplifies the merging of Java based tests into your global testing ecosystem.
