# 1. Installing Testkube CLI

To install Testkube CLI you'll need the following tools:

- [Kubectl](https://kubernetes.io/docs/tasks/tools/), Kubernetes command-line tool
- [Helm](https://helm.sh/)

:::note 

Installing the Testkube CLI with Chocolatey and Homebrew will automatically install these dependencies if they are not present. For Linux-based systems please install them manually in advance.

::: 

#### MacOS

```bash
brew install testkube
```

#### Windows

```bash
choco source add --name=kubeshop_repo --source=https://chocolatey.kubeshop.io/chocolatey  
choco install testkube -y
```

#### Ubuntu/Debian

```bash
wget -qO - https://repo.testkube.io/key.pub | sudo apt-key add - && echo "deb https://repo.testkube.io/linux linux main" | sudo tee -a /etc/apt/sources.list && sudo apt-get update && sudo apt-get install -y testkube
```

#### Script Installation

```bash
curl -sSLf https://get.testkube.io | sh
```

#### Manual Download

If you don't want to use scripts or package managers you can always do a manual install:

1. Download the binary for the version and platform of your choice [here](https://github.com/kubeshop/testkube/releases)
2. Unpack it. For example, in Linux use `tar -zxvf testkube_1.5.1_Linux_x86_64.tar.gz`
3. Move it to a location in the PATH. For example:

```sh
mv  testkube_1.5.1_Linux_x86_64/kubectl-testkube /usr/local/bin/kubectl-testkube`
```

For Windows, you will need to unpack the binary and add it to the `%PATH%` as well.

:::note
If you use a package manager that we don't support, please let us know here [#161](https://github.com/kubeshop/testkube/issues/161).
:::