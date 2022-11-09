---
sidebar_position: 1
sidebar_label: Contributing
---
# Contributing to Projects

If you are new to the open source community, use this guide to start contributing to projects:
<https://github.com/firstcontributions/first-contributions>.

Checkout the [development document](development/developments.md) for more details about how to develop and run testkube on your machine.

## **General Guidance for Contributing to a Testkube Project**

Anyone is welcome and encouraged to help in Testkube development; much opportunity for enhancement exists.

We would like to limit technical debt from the beginning, so we have defined simple rules when adding code into Testkube repo.

### **For Go Programming Language (Golang) Based Components**

- Always use gofmt.
- Follow Golang good practices ([proverbs](https://go-proverbs.github.io/)) in your code.
- Testing is your friend. We will target 80% CC in our code.
- Use clean names and don't break basic design patterns and rules.

### **For Infrastructure/Kubernetes Based Components**

- Put in comments for non-obvious decisions.
- Use current Helm/Kubernetes versions.

## **Building Diagrams**

To build diagrams, install PlantUML:

```bash
brew install plantuml # on mac
```

```bash
sudo apt-get install -y plantuml # on ubuntu linux 
```

```bash
pacman -S plantuml # on arch linux
```

Then run:

```bash
make diagrams
```

This generates png files from puml files.

TIP: If using vscode, there is a nice extension for the live preview of PlantUML files.  
