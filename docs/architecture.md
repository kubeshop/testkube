# Architecture (C4 Diagrams)

## **Info**

This diagram was made with the C4 diagram technique
(<https://c4model.com/>).

## **Diagrams**

### **System Context**

![testkube system context diagram](img/system_context.png)

### **Containers**

![testkube container diagram](img/containers.png)

### **Components**

#### **API**

![API](img/components_api.png)

### TestSuites and Tests

Params passing through tests suites and tests priority: 

1. Test suite execution variables overrides
2. Test suite variables overrides
3. Test execution (variables passed for single test runs) overrides
4. Test variables


![variables passing](img/params_passing.png)
