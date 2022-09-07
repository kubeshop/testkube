---
sidebar_position: 10
sidebar_label: Architecture
---
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

Parameters passed through tests suites and tests priority: 

1. Test Suite execution variables overrides.
2. Test Suite variables overrides.
3. Test execution (variables passed for single test runs) overrides.
4. Test variables.


![variables passing](img/params-passing.png)
