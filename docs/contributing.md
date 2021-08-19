# Contribution to project 

If you're new in Open-source community there is nice guide how to start contributing to projects: 
https://github.com/firstcontributions/first-contributions

## General guidance for contributing to kubtest project

You're very welcome to help in kubtest development, there is a lot of incoming work to do :). 

We're trying hard to limit technical debt from the beginning so we defined simple rules when putting some code into kubtest repo.

### For golang based components

- Always use gofmt
- Follow golang good practices (proverbs) in your code
- Tests are your friend (we will target 80% CC in our code)
- Use clean names, don't brake basic design patterns and rules.

### For infrastructure / Kubernetes based components

- Comment non-obvious decisions
- Use current Helm/Kubernetes versions

## Building diagrams

To build diagrams you'll need to install plantuml:

```sh
brew install plantuml # on mac
sudo apt-get install -y plantuml # on ubuntu linux 
pacman -S plantuml # on arch linux
```
Next run 
```sh 
make diagrams
```
to generate png files from puml files

TIP: If using vscode there is nice extension for live preview of plantuml files.  
