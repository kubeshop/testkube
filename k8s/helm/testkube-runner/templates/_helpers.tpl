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
{{- $separator := ":" -}}
{{- if .Values.images.agent.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.agent.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag -}}
    {{- else -}}
        {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
{{- end -}}
{{- end }}

{{- define "testkube-runner.toolkit.image" -}}
{{- $registryName := .Values.images.toolkit.registry -}}
{{- $repositoryName := .Values.images.toolkit.repository -}}
{{- $tag := default .Chart.AppVersion .Values.images.toolkit.tag | toString -}}
{{- $separator := ":" -}}
{{- if .Values.images.toolkit.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.toolkit.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag -}}
    {{- else -}}
        {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
{{- end -}}
{{- end }}

{{- define "testkube-runner.init.image" -}}
{{- $registryName := .Values.images.init.registry -}}
{{- $repositoryName := .Values.images.init.repository -}}
{{- $tag := default .Chart.AppVersion .Values.images.init.tag | toString -}}
{{- $separator := ":" -}}
{{- if .Values.images.init.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.images.init.digest | toString -}}
{{- end -}}
{{- if .Values.global }}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag -}}
    {{- else -}}
        {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s" $registryName $repositoryName $separator $tag -}}
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
