# Why Use Testkube?

## **Streamline Disperate Testing Platforms**

### **Problem**
A large IT department has a frontend team and a backend team, everything is deployed on Kubernetes cluster, and each team is responsible for its part of the work. The frontend engineers test their code using the Cypress testing framework, but the backend engineers prefer simpler tools like Postman. They have a lot of Postman collections defined and want to run them against a Kubernetes cluster but some of their services are not exposed externally.

A QA leader is responsible for production releases and wants to be sure that all tests are completed successfully. The QA leader will need to create pipelines that orchestrate each teams' tests into a common platform.

### **Solution**
This is easily done with Testkube. Each team can run their tests against clusters on their own, and the QA manager can create test resources and add tests written by all teams.

Test Suites stands for the orchestration of different test steps such as test execution, delay, or other (future) steps.

### **Results**
Not requiring retraining of staff to all use the same platform for testing and allowing for different types of tests to be run together to facilitate the required testing flow speeds up the development/deployment process, saving a company both time and money.