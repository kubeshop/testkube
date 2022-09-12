---
sidebar_position: 8
sidebar_label: Gradle
---
# Gradle Based Tests

Testkube allows running Gradle based tasks that could be also tests. For example, we can easily run JUnit tests in Testkube. 


## **Test Environment**

We will put a simple JUnit test in our cluster and run it. The Testkube Gradle Executor handles `gradle` and `gradlew` binaries.
Since Gradle projects are quite complicated in terms of directory structure, we'll need to load them from the Git directory.

You can find example projects in the repository here: https://github.com/kubeshop/testkube-executor-gradle/tree/main/examples.

Let's create a simple test which will check if an env variable is set to true: 
```java
package hello.gradle;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class LibraryTest {
    @Test void someLibraryMethodReturnsTrue() {
        String env = System.getenv("TESTKUBE_GRADLE");
        assertTrue(Boolean.parseBoolean(env), "TESTKUBE_GRADLE env should be true");
    }
}
```


The default Gradle executor looks like: 

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: gradle-jdk18-executor
  namespace: testkube
spec:
  image: kubeshop/testkube-gradle-executor:0.1.4-jdk18
  types:
  - gradle/project
  - gradle/test
  - gradle/integrationTest 
```

As we can see, there are several types. The Gradle executor handles the second part after `/` as a task name, so `gradle/test` will run `gradle test` and so on. 

As opposed to `project` which is generic and forces you to pass additional arguments during test execution. 
For example:

```bash
kubectl testkube run gradle-example-project --args='runMyCustomTask' 
```


## **Create a New Gradle-based Test**

```bash
kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-gradle.git --git-path examples/hello-gradle --type gradle/test --name gradle-example-test --git-branch main
```



## **Running a Test**

Let's pass the env variable to our test run:

```bash
 kubectl testkube run test gradle-example-test -f -v TESTKUBE_GRADLE=true

# ...... after some time

The test execution completed successfully in 16.555s 🥇.

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
# ....... a lot of Gradle logs

> Task :compileTestJava
> Task :processTestResources NO-SOURCE
> Task :testClasses
> Task :test

BUILD SUCCESSFUL in 10s
2 actionable tasks: 2 executed

Status Test execution completed with success 🥇
```

## Using Different JDKs 

In the Java world, you would like to have control over the Runtime environment. Testkube can easily handle that for you! 
We're building several Java images to handle constraints which Gradle can put in its build file.

To use a different executor, you can use one of our pre-built ones (for Java 8,11,17,18) or build your own Docker image based on the Gradle executor.

Let's assume we need JDK18 for our test runs. In Testkube, create a new Gradle executor:

content of `gradle-jdk18-executor.yaml`
```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: gradle-jdk18-executor
  namespace: testkube
spec:
  image: kubeshop/testkube-gradle-executor:0.1.4-jdk18
  types:
  - gradle:jdk18/project
  - gradle:jdk18/test
  - gradle:jdk18/integrationTest 
```

And add it to your cluster: 
```bash
kubectl apply -f gradle-jdk18-executor.yaml 
```

Now, create the new test with the type that our new executor can handle e.g.: `gradle:jdk18/test`:

```bash 
 # create test:
 kubectl testkube create test --git-uri https://github.com/kubeshop/testkube-executor-gradle.git --git-path examples/hello-gradle-jdk18 --type gradle:jdk18/test --name gradle-jdk18-example-test --git-branch main

# and run it:
kubectl testkube run test gradle-jdk18-example-test -f -v TESTKUBE_GRADLE=true
```


## **Summary**

Testkube simplifies running Java/Kotlin based tests (`build.gradle.kts` is also handled) and allows merging Java based tests into your global testing ecosystem easily.
