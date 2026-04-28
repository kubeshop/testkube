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
