{{- define "testkube-api.executors" -}}
[
  {
    "name": "tracetest-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-tracetest-executor:{{ .Values.image.executorsTag }}",
      "command": [
        "tracetest"
      ],
      "args": [
        "test",
        "run",
        "--server-url",
        "<tracetestServer>",
        "--definition",
        "<filePath>",
        "--wait-for-result",
        "--output",
        "pretty"
      ],
      "types": [
        "tracetest/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "tracetest",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-tracetest"
      }
    }
  },
  {
    "name": "zap-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-zap-executor:{{ .Values.executorsTag }}",
      "command": [
        "<pythonScriptPath>"
      ],
      "args": [
        "<fileArgs>"
      ],
      "types": [
        "zap/api",
        "zap/baseline",
        "zap/full"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "zap",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-zap"
      }
    }
  },
  {
    "name": "playwright-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-playwright-executor:{{ .Values.executorsTag }}",
      "command": [
        "<depManager>"
      ],
      "args": [
        "<depCommand>",
        "playwright",
        "test"
      ],
      "types": [
        "playwright/test"
      ],
      "contentTypes": [
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "playwright",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-playwright"
      }
    }
  },
  {
    "name": "jmeter-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-jmeter-executor:{{ .Values.executorsTag }}",
      "command": [
        "<entryPoint>"
      ],
      "args": [
        "-n",
        "-j",
        "<logFile>",
        "-t",
        "<runPath>",
        "-l",
        "<jtlFile>",
        "-o",
        "<reportFile>",
        "-e",
        "<envVars>"
      ],
      "types": [
        "jmeter/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "jmeter",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-jmeter"
      }
    }
  },
  {
    "name": "jmeterd-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-jmeterd-executor:{{ .Values.executorsTag }}",
      "command": [
        "<entryPoint>"
      ],
      "slaves": {
        "image": "kubeshop/testkube-jmeterd-slave:{{ .Values.executorsTag }}"
      },
      "args": [
        "-n",
        "-j",
        "<logFile>",
        "-t",
        "<runPath>",
        "-l",
        "<jtlFile>",
        "-o",
        "<reportFile>",
        "-e",
        "<envVars>"
      ],
      "types": [
        "jmeterd/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "jmeter",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-jmeter"
      }
    }
  },
  {
    "name": "ginkgo-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-ginkgo-executor:{{ .Values.executorsTag }}",
      "command": [
        "ginkgo"
      ],
      "args": [
        "-r",
        "-p",
        "--randomize-all",
        "--randomize-suites",
        "--keep-going",
        "--trace",
        "--junit-report",
        "<reportFile>",
        "<envVars>",
        "<runPath>"
      ],
      "types": [
        "ginkgo/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts",
        "junit-report"
      ],
      "meta": {
        "iconURI": "ginkgo",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-ginkgo"
      }
    }
  },
  {
    "name": "maven-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-maven-executor:{{ .Values.executorsTag }}",
      "command": [
        "mvn"
      ],
      "args": [
        "--settings",
        "<settingsFile>",
        "<goalName>",
        "-Duser.home",
        "<mavenHome>"
      ],
      "types": [
        "maven/project",
        "maven/test",
        "maven/integration-test"
      ],
      "contentTypes": [
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "maven",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-maven"
      }
    }
  },
  {
    "name": "gradle-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-gradle-executor:{{ .Values.executorsTag }}",
      "command": [
        "gradle"
      ],
      "args": [
        "--no-daemon",
        "<taskName>",
        "-p",
        "<projectDir>"
      ],
      "types": [
        "gradle/project",
        "gradle/test",
        "gradle/integrationTest"
      ],
      "contentTypes": [
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "gradle",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-gradle"
      }
    }
  },
  {
    "name": "kubepug-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-kubepug-executor:{{ .Values.executorsTag }}",
      "command": [
        "kubepug"
      ],
      "args": [
        "--format=json",
        "--input-file",
        "<runPath>"
      ],
      "types": [
        "kubepug/yaml",
        "kubepug/json"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "kubepug",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-kubepug"
      }
    }
  },
  {
    "name": "soapui-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-soapui-executor:{{ .Values.executorsTag }}",
      "command": [
        "/bin/sh",
        "/usr/local/SmartBear/EntryPoint.sh"
      ],
      "args": [
        "<runPath>"
      ],
      "types": [
        "soapui/xml"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "soapui",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-soapui"
      }
    }
  },
  {
    "name": "k6-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-k6-executor:{{ .Values.executorsTag }}",
      "command": [
        "k6"
      ],
      "args": [
        "<k6Command>",
        "<envVars>",
        "<runPath>"
      ],
      "types": [
        "k6/script"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "k6",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-k6"
      }
    }
  },
  {
    "name": "cypress-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-cypress-executor:{{ .Values.executorsTag }}",
      "command": [
        "./node_modules/cypress/bin/cypress"
      ],
      "args": [
        "run",
        "--reporter",
        "junit",
        "--reporter-options",
        "mochaFile=<reportFile>,toConsole=false",
        "--project",
        "<projectPath>",
        "--env",
        "<envVars>"
      ],
      "types": [
        "cypress/project"
      ],
      "contentTypes": [
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts",
        "junit-report"
      ],
      "meta": {
        "iconURI": "cypress",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-cypress"
      }
    }
  },
  {
    "name": "curl-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-curl-executor:{{ .Values.executorsTag }}",
      "command": [
        "curl"
      ],
      "args": [
        "-is"
      ],
      "types": [
        "curl/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "curl",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-curl"
      }
    }
  },
  {
    "name": "postman-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-postman-executor:{{ .Values.executorsTag }}",
      "command": [
        "newman"
      ],
      "args": [
        "run",
        "<runPath>",
        "-e",
        "<envFile>",
        "--reporters",
        "cli,json",
        "--reporter-json-export",
        "<reportFile>"
      ],
      "types": [
        "postman/collection"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "postman",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-postman"
      }
    }
  },
  {
    "name": "artillery-executor",
    "executor": {
      "executorType": "job",
      "image": "kubeshop/testkube-artillery-executor:{{ .Values.executorsTag }}",
      "command": [
        "artillery"
      ],
      "args": [
        "run",
        "<runPath>",
        "--dotenv",
        "<envFile>",
        "-o",
        "<reportFile>"
      ],
      "types": [
        "artillery/test"
      ],
      "contentTypes": [
        "string",
        "file-uri",
        "git-file",
        "git-dir",
        "git"
      ],
      "features": [
        "artifacts"
      ],
      "meta": {
        "iconURI": "artillery",
        "docsURI": "https://kubeshop.github.io/testkube/test-types/executor-artillery"
      }
    }
  },
  {
    "name": "scraper-executor",
    "executor": {
      "executorType": "scraper",
      "image": "kubeshop/testkube-scraper-executor:{{ .Values.executorsTag }}",
      "types": []
    }
  },
  {
    "name": "init-executor",
    "executor": {
      "executorType": "init",
      "image": "kubeshop/testkube-init-executor:{{ .Values.executorsTag }}",
      "types": []
    }
  },
  {
    "name": "logs-sidecar",
    "executor": {
      "executorType": "sidecar",
      "image": "kubeshop/testkube-logs-sidecar:{{ .Values.executorsTag }}",
      "types": []
    }
  }
]
{{- end }}