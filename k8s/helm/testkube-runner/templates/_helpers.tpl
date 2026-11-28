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

{{- define "testkube-runner.runnerName" -}}
{{- default (include "testkube-runner.fullname" .) .Values.runner.name -}}
{{- end }}

{{- define "testkube-runner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "testkube-runner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "testkube-runner.labels" -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{ include "global.labels.standard" . }}
{{ include "testkube-runner.selectorLabels" . }}
{{- end }}

{{/*
Monitoring labels
*/}}
{{- define "testkube-runner.monitoring" -}}
app: prometheus
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
    {{- if .Values.global.testkubeVersion -}}
        {{- $tag = .Values.global.testkubeVersion | toString -}}
    {{- end -}}
    {{- if .Values.global.imageRegistry }}
        {{- printf "%s/%s%s%s%s" .Values.global.imageRegistry $repositoryName $separator $tag $tagSuffix -}}
    {{- else -}}
        {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
    {{- end -}}
{{- else -}}
    {{- printf "%s/%s%s%s%s" $registryName $repositoryName $separator $tag $tagSuffix -}}
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
    {{- if .Values.global.testkubeVersion -}}
        {{- $tag = .Values.global.testkubeVersion | toString -}}
    {{- end -}}
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
    {{- if .Values.global.testkubeVersion -}}
        {{- $tag = .Values.global.testkubeVersion | toString -}}
    {{- end -}}
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

{{/*
Value of GITOPS_NAMESPACES. Each entry is a namespace, a namespace=<Environment id>, or * for all namespaces.
The * goes first, so the Environment ids of a map still apply to their namespaces when all namespaces are watched.
*/}}
{{- define "testkube-runner.gitops.namespaces" -}}
{{- $entries := list -}}
{{- if .Values.gitops.watchAllNamespaces -}}
{{- $entries = append $entries "*" -}}
{{- end -}}
{{- $namespaces := default (list) .Values.gitops.namespaces -}}
{{- if kindIs "map" $namespaces -}}
{{- range $name := keys $namespaces | sortAlpha -}}
{{- $env := get $namespaces $name | default "" | toString -}}
{{- if $env -}}
{{- $entries = append $entries (printf "%s=%s" $name $env) -}}
{{- else -}}
{{- $entries = append $entries $name -}}
{{- end -}}
{{- end -}}
{{- else -}}
{{- range $name := $namespaces -}}
{{- $entries = append $entries (toString $name) -}}
{{- end -}}
{{- end -}}
{{- join "," $entries -}}
{{- end }}

{{- define "testkube-runner.gitops.namespaceNames" -}}
{{- if .Values.gitops.enabled -}}
{{- $namespaces := default (list) .Values.gitops.namespaces -}}
{{- if kindIs "map" $namespaces -}}
{{- join "," (keys $namespaces | sortAlpha) -}}
{{- else -}}
{{- join "," $namespaces -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Whether the GitOps runner syncs the Control Plane resources: "true", or empty. Values reused from a
release made before gitops.controlPlaneResources existed (helm upgrade --reuse-values) lack it, so
it is read with a default rather than directly.
*/}}
{{- define "testkube-runner.gitops.controlPlaneResources" -}}
{{- if and .Values.gitops.enabled (get (default dict .Values.gitops.controlPlaneResources) "enabled") -}}
true
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
      - testworkflowtemplates
      - workflowtriggers
    verbs:
      - get
      - list
      - watch
  {{- if include "testkube-runner.gitops.controlPlaneResources" . }}
  # Control Plane resources (Insights Boards, AI Agents, MCP servers, LLM providers and Skills),
  # which the GitOps runner syncs to the Control Plane. It only reads them, and the ConfigMaps
  # above that a Skill's skillFrom reads its SKILL.md from.
  - apiGroups:
      - "insights.testkube.io"
    resources:
      - boards
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "ai.testkube.io"
    resources:
      - aiagents
      - mcpservers
      - llmproviders
      - skills
    verbs:
      - get
      - list
      - watch
  {{- end }}
  {{- with .Values.listener.extraWatchedResources }}
  # User-declared GVKs for TestTrigger / WorkflowTrigger `resourceRef` targets
  # that aren't covered by the built-in resource list above.
  {{- range . }}
  - apiGroups: {{ toYaml .apiGroups | nindent 6 }}
    resources: {{ toYaml .resources | nindent 6 }}
    verbs: {{ .verbs | default (list "get" "list" "watch") | toYaml | nindent 6 }}
  {{- end }}
  {{- end }}
{{- end }}
