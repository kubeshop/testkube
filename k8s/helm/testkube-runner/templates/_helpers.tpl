{{- define "testkube-runner.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "testkube-runner.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "testkube-runner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "testkube-runner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "testkube-runner.agent.image" -}}
{{- $registryName := .Values.images.agent.registry -}}
{{- $repositoryName := .Values.images.agent.repository -}}
{{- $tag := default .Chart.AppVersion .Values.images.agent.tag | toString -}}
{{- $tagSuffix := .Values.images.agent.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.images.agent.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.agent.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag $tagsuffix -}}
    {{- else -}}
        {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagsuffix -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagsuffix -}}
{{- end -}}
{{- end }}

{{- define "testkube-runner.toolkit.image" -}}
{{- $registryName := .Values.images.toolkit.registry -}}
{{- $repositoryName := .Values.images.toolkit.repository -}}
{{- $tag := default .Chart.AppVersion .Values.images.toolkit.tag | toString -}}
{{- $tagSuffix := .Values.images.toolkit.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.images.toolkit.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.toolkit.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag $tagSuffix -}}
    {{- else -}}
        {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
{{- end -}}
{{- end }}

{{- define "testkube-runner.init.image" -}}
{{- $registryName := .Values.images.init.registry -}}
{{- $repositoryName := .Values.images.init.repository -}}
{{- $tag := default .Chart.AppVersion .Values.images.init.tag | toString -}}
{{- $tagSuffix := .Values.images.init.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.images.init.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.init.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag $tagSuffix -}}
    {{- else -}}
        {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
{{- end -}}
{{- end }}

{{- define "testkube-runner.containerSecurityContext" -}}
{{- if .Values.global.containerSecurityContext }}
{{- toYaml .Values.global.containerSecurityContext }}
{{- else }}
{{- toYaml .Values.pod.containerSecurityContext }}
{{- end }}
{{- end }}

{{- define "testkube-runner.podSecurityContext" -}}
{{- if .Values.global.podSecurityContext }}
{{ toYaml .Values.global.podSecurityContext }}
{{- else }}
{{ toYaml .Values.pod.securityContext }}
{{- end }}
{{- end }}

{{- define "testkube-runner.eventLabels" -}}
{{- $yamlString := toYaml .Values.listener.eventLabels }}
{{- $lines := split "\n" (trim $yamlString) }}
{{- $processedLines := list }}
{{- range $line := $lines }}
{{- $processedLines = append $processedLines (regexReplaceAll ": " $line ":") }}
{{- end }}
{{- join "," $processedLines -}}
{{- end }}

{{/*
Define TESTKUBE_WATCHER_NAMESPACES variable
*/}}
{{- define "testkube-runner.watcher-namespaces" -}}
{{- if .Values.listener.watchAllNamespaces -}}
{{- "" -}}
{{- else -}}
{{- $additional := default (list) .Values.listener.additionalNamespaces -}}
{{ join "," (concat (list .Release.Namespace) $additional) }}
{{- end -}}
{{- end }}

{{- define "testkube-runner.listener.watchers.rules" -}}
rules:
  - apiGroups:
      - ""
    resources:
      - pods
      - services
      - events
      - namespaces
      - configmaps
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "apps"
    resources:
      - deployments
      - daemonsets
      - statefulsets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "networking.k8s.io"
      - "extensions"
    resources:
      - ingresses
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "tests.testkube.io"
    resources:
      - testtriggers
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "executor.testkube.io"
    resources:
      - webhooks
      - webhooktemplates
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "testworkflows.testkube.io"
    resources:
      - testworkflows
    verbs:
      - get
      - list
      - watch
{{- end }}
