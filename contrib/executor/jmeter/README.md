![Testkube Logo](https://raw.githubusercontent.com/kubeshop/testkube/main/assets/testkube-color-gray.png)

# Welcome to TestKube JMeter Executor

It's basic JMeter executor able to run simple JMeter scenarios writter in JMX format. Please define your JMeter file as file (string, or git file). 
Project directory is not implemented yet.

# What is an Executor?

Executor is nothing more than a program wrapped into Docker container which gets JSON (testube.Execution) OpenAPI based document as an input and returns a stream of JSON output lines (testkube.ExecutorOutput), where each output line is simply wrapped in this JSON, similar to the structured logging idea. 


# Issues and enchancements 

Please follow the main [TestKube repository](https://github.com/kubeshop/testkube) for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)
