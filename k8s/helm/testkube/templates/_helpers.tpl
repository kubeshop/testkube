{{/*
MongoDB upgrade labels
*/}}
{{- define "mongodb.labels" -}}
app.kubernetes.io/component: mongodb
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: mongodb-upgrade
{{- end -}}

{{/*
Mongo FCV Job labels
*/}}
{{- define "testkube.mongoFcv.labels" -}}
{{- include "global.labels.standard" . }}
app.kubernetes.io/component: mongodb-fcv
app.kubernetes.io/version: {{ .Values.mongodb.image.tag | quote }}
{{ include "testkube.mongoFcv.selectorLabels" . }}
{{- if .Values.global.labels }}
{{ toYaml .Values.global.labels }}
{{- end }}
{{- end }}

{{/*
Mongo FCV selector labels
*/}}
{{- define "testkube.mongoFcv.selectorLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: mongodb-fcv
{{- end }}

{{/*
Mongo FCV prepare Job name
*/}}
{{- define "testkube.mongoFcv.prepareName" -}}
{{- $name := .Values.mongodb.fullnameOverride | default (printf "%s-mongodb" .Release.Name) -}}
{{- printf "%s-fcv-prepare-%d" $name .Release.Revision | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Mongo FCV apply Job name
*/}}
{{- define "testkube.mongoFcv.applyName" -}}
{{- $name := .Values.mongodb.fullnameOverride | default (printf "%s-mongodb" .Release.Name) -}}
{{- printf "%s-fcv-apply-%d" $name .Release.Revision | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Mongo FCV Job podSecurityContext
*/}}
{{- define "testkube.mongoFcv.podSecurityContext" -}}
{{- $secContext := dict -}}
{{- if .Values.global.podSecurityContext }}
{{- $secContext = .Values.global.podSecurityContext -}}
{{- else if .Values.mongodb.preUpgradeFCVJob.podSecurityContext }}
{{- $secContext = .Values.mongodb.preUpgradeFCVJob.podSecurityContext -}}
{{- end }}
{{- if hasKey $secContext "enabled" }}
{{ omit $secContext "enabled" | toYaml }}
{{- else }}
{{ toYaml $secContext }}
{{- end }}
{{- end }}

{{/*
Mongo FCV Job containerSecurityContext
*/}}
{{- define "testkube.mongoFcv.containerSecurityContext" -}}
{{- $secContext := dict -}}
{{- if .Values.global.containerSecurityContext }}
{{- $secContext = .Values.global.containerSecurityContext -}}
{{- else if .Values.mongodb.preUpgradeFCVJob.securityContext }}
{{- $secContext = .Values.mongodb.preUpgradeFCVJob.securityContext -}}
{{- end }}
{{- if hasKey $secContext "enabled" }}
{{ omit $secContext "enabled" | toYaml }}
{{- else }}
{{ toYaml $secContext }}
{{- end }}
{{- end }}

{{/*
Mongo FCV Job nodeSelector
*/}}
{{- define "testkube.mongoFcv.nodeSelector" -}}
{{- if .Values.mongodb.preUpgradeFCVJob.nodeSelector }}
{{ toYaml (.Values.mongodb.preUpgradeFCVJob.nodeSelector | default dict) }}
{{- else if .Values.global.nodeSelector }}
{{ toYaml .Values.global.nodeSelector }}
{{- end }}
{{- end }}

{{/*
Mongo FCV Job affinity
*/}}
{{- define "testkube.mongoFcv.affinity" -}}
{{- if .Values.mongodb.preUpgradeFCVJob.affinity }}
{{ toYaml (.Values.mongodb.preUpgradeFCVJob.affinity | default dict) }}
{{- else if .Values.global.affinity }}
{{ toYaml .Values.global.affinity }}
{{- end }}
{{- end }}

{{/*
Mongo FCV Job tolerations
*/}}
{{- define "testkube.mongoFcv.tolerations" -}}
{{- if .Values.mongodb.preUpgradeFCVJob.tolerations }}
{{ toYaml .Values.mongodb.preUpgradeFCVJob.tolerations }}
{{- else if .Values.global.tolerations }}
{{ toYaml .Values.global.tolerations }}
{{- else }}
{{ toYaml .Values.mongodb.tolerations }}
{{- end }}
{{- end }}

{{/*
PostgreSQL upgrade labels
*/}}
{{- define "postgresql.labels" -}}
app.kubernetes.io/component: postgresql
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: postgresql-upgrade
{{- end -}}

{{/*
Convert Job name. Suffixed with the release revision so that re-running the
migration on a later upgrade creates a new Job instead of colliding with the
completed one, whose pod template is immutable.
*/}}
{{- define "testkube.convert.name" -}}
{{- printf "%s-convert-%d" .Release.Name .Release.Revision | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Convert Job labels
*/}}
{{- define "testkube.convert.labels" -}}
{{- include "global.labels.standard" . }}
app.kubernetes.io/component: convert
{{ include "testkube.convert.selectorLabels" . }}
{{- if .Values.global.labels }}
{{ toYaml .Values.global.labels }}
{{- end }}
{{- end }}

{{/*
Convert Job selector labels
*/}}
{{- define "testkube.convert.selectorLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: testkube-convert
{{- end }}

{{/*
Convert Job MongoDB environment.

Sourced from the testkube-api subchart values rather than from a dedicated
convert block, so the DSN and its TLS material are never configured twice. Note
this is deliberately NOT gated on testkube-api.mongodb.enabled: the whole point
of the Job is to migrate off MongoDB, so it must still be able to read from it
once the API has been pointed at PostgreSQL.
*/}}
{{- define "testkube.convert.mongoEnv" -}}
{{- $mongo := (index .Values "testkube-api").mongodb -}}
- name: API_MONGO_DSN
  {{- if $mongo.secretName }}
  valueFrom:
    secretKeyRef:
      name: {{ $mongo.secretName }}
      key: {{ required "testkube-api.mongodb.secretKey is required when secretName is set" $mongo.secretKey }}
  {{- else }}
  value: {{ required "testkube-api.mongodb.dsn is required to run the convert Job" $mongo.dsn | quote }}
  {{- end }}
{{- with $mongo.dbType }}
- name: API_MONGO_DB_TYPE
  value: {{ . | quote }}
{{- end }}
{{- with $mongo.allowTLS }}
- name: API_MONGO_ALLOW_TLS
  value: {{ . | quote }}
{{- end }}
{{- end }}

{{/*
Convert Job PostgreSQL environment.
*/}}
{{- define "testkube.convert.postgresEnv" -}}
{{- $postgres := (index .Values "testkube-api").postgresql -}}
- name: API_POSTGRES_DSN
  {{- if $postgres.secretName }}
  valueFrom:
    secretKeyRef:
      name: {{ $postgres.secretName }}
      key: {{ required "testkube-api.postgresql.secretKey is required when secretName is set" $postgres.secretKey }}
  {{- else }}
  value: {{ required "testkube-api.postgresql.dsn is required to run the convert Job" $postgres.dsn | quote }}
  {{- end }}
{{- end }}

{{/*
Convert Job image. Mirrors testkube-api.image: the tag falls back to the chart
appVersion, and global.testkubeVersion / global.imageRegistry win when set, so
the Job is always built from the same Testkube release as the API.
*/}}
{{- define "testkube.convert.image" -}}
{{- $image := .Values.convert.image -}}
{{- $registryName := $image.registry -}}
{{- $tag := default .Chart.AppVersion $image.tag | toString -}}
{{- $separator := ":" -}}
{{- if $image.digest }}
    {{- $separator = "@" -}}
    {{- $tag = $image.digest | toString -}}
{{- else if .Values.global }}
    {{- if .Values.global.testkubeVersion }}
        {{- $tag = .Values.global.testkubeVersion | toString -}}
    {{- end }}
{{- end }}
{{- if and .Values.global .Values.global.imageRegistry }}
    {{- $registryName = .Values.global.imageRegistry -}}
{{- end }}
{{- printf "%s/%s%s%s" $registryName $image.repository $separator $tag -}}
{{- end }}
