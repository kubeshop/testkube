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
PostgreSQL upgrade labels
*/}}
{{- define "postgresql.labels" -}}
app.kubernetes.io/component: postgresql
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: postgresql-upgrade
{{- end -}}