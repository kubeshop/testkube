{{/*
Expand the name of the chart.
*/}}
{{- define "testkube-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "testkube-api.fullname" -}}
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

{{/*
API labels
*/}}
{{- define "testkube-api.labels" -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{ include "global.labels.standard" . }}
{{ include "testkube-api.selectorLabels" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "testkube-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "testkube-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Monitoring labels
*/}}
{{- define "testkube-api.monitoring" -}}
app: prometheus
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "testkube-api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "testkube-api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the test service account to use
*/}}
{{- define "testkube-api.testServiceAccountName" -}}
{{- if .Values.testServiceAccount.create }}
{{- $prefix := default (include "testkube-api.fullname" .) .Values.jobServiceAccountName }}
{{- printf "%s-%s" $prefix "tests-job" }}
{{- else }}
{{- default "default" .Values.jobServiceAccountName }}
{{- end }}
{{- end }}

{{/*
Define API image
*/}}
{{- define "testkube-api.image" -}}
{{- $registryName := .Values.image.registry -}}
{{- $repositoryName := .Values.image.repository -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag | toString -}}
{{- $tagSuffix := .Values.image.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.image.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.image.digest | toString -}}
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
{{- end -}}

{{/*
Define API environment in agent mode
*/}}
{{- define "testkube-api.env-agent-mode" -}}
{{- if .Values.cloud.key -}}
- name: TESTKUBE_PRO_API_KEY
  value:  "{{ .Values.cloud.key }}"
- name: TESTKUBE_PRO_AGENT_ID
  value: "{{ .Values.cloud.agentId }}"
{{- else if .Values.cloud.existingSecret.key -}}
- name: TESTKUBE_PRO_API_KEY
  valueFrom:
    secretKeyRef:
      key: {{ .Values.cloud.existingSecret.key }}
      name: {{ .Values.cloud.existingSecret.name }}
{{- end }}
- name: RUNNER_IS_GLOBAL
  value: "true"
{{- if .Values.cloud.url }}
- name: TESTKUBE_CLOUD_URL
  value:  {{ tpl .Values.cloud.url $ | quote }}
- name: TESTKUBE_PRO_URL
  value:  {{ tpl .Values.cloud.url $ | quote }}
{{- end }}
{{- if .Values.cloud.uiUrl}}
- name: TESTKUBE_CLOUD_UI_URL
  value: {{ tpl .Values.cloud.uiUrl $ | quote }}
- name: TESTKUBE_PRO_UI_URL
  value: {{ tpl .Values.cloud.uiUrl $ | quote }}
{{- end}}
{{- if not .Values.cloud.tls.enabled }}
- name: TESTKUBE_PRO_TLS_INSECURE
  value:  "true"
{{- end }}
{{- if .Values.cloud.tls.certificate.secretRef }}
- name: TESTKUBE_PRO_TLS_SECRET
  value: {{ .Values.cloud.tls.certificate.secretRef }}
- name: TESTKUBE_PRO_CERT_FILE
  value:  {{ .Values.cloud.tls.certificate.certFile }}
- name: TESTKUBE_PRO_KEY_FILE
  value: {{ .Values.cloud.tls.certificate.keyFile }}
- name: TESTKUBE_PRO_CA_FILE
  value: {{ .Values.cloud.tls.certificate.caFile }}
{{- end }}
- name: TESTKUBE_PRO_SKIP_VERIFY
  value:  "{{ if hasKey .Values.global.tls "skipVerify" }}{{ .Values.global.tls.skipVerify }}{{ else }}{{ .Values.cloud.tls.skipVerify }}{{ end }}"
{{- if .Values.cloud.orgId }}
- name: TESTKUBE_PRO_ORG_ID
  value:  "{{ .Values.cloud.orgId }}"
{{- end}}
{{- if .Values.cloud.existingSecret.orgId }}
- name: TESTKUBE_PRO_ORG_ID
  valueFrom:
    secretKeyRef:
      key: {{ .Values.cloud.existingSecret.orgId }}
      name: {{ .Values.cloud.existingSecret.name }}
{{- end}}
{{- if .Values.cloud.envId }}
- name: TESTKUBE_PRO_ENV_ID
  value:  "{{ .Values.cloud.envId }}"
{{- end}}
{{- if .Values.cloud.existingSecret.envId }}
- name: TESTKUBE_PRO_ENV_ID
  valueFrom:
    secretKeyRef:
      key: {{ .Values.cloud.existingSecret.envId }}
      name: {{ .Values.cloud.existingSecret.name }}
{{- end}}
{{- if .Values.cloud.migrate }}
- name: TESTKUBE_PRO_MIGRATE
  value:  "{{ .Values.cloud.migrate }}"
{{- end}}
- name: "NATS_EMBEDDED"
  value: "{{ .Values.nats.embedded }}"
- name: NATS_URI
  {{- if .Values.nats.uri }}
  value: {{ .Values.nats.uri }}
  {{- else if .Values.nats.secretName }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.nats.secretName }}
      key: {{ .Values.nats.secretKey }}
  {{- else }}
  value: "nats://{{ .Release.Name }}-nats"
  {{- end }}
- name: "NATS_SECURE"
  value: "{{ .Values.nats.tls.enabled }}"
{{- if .Values.nats.tls.certSecret.enabled }}
- name: "NATS_CERT_FILE"
  value:  "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.certFile }}"
- name: "NATS_KEY_FILE"
  value: "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.keyFile }}"
{{- if .Values.nats.tls.mountCACertificate }}
- name: "NATS_CA_FILE"
  value: "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.caFile }}"
{{- end }}
{{- end }}
- name: "SCRAPPERENABLED"
  value:  "{{ .Values.storage.scrapperEnabled }}"
- name: "COMPRESSARTIFACTS"
  value:  "{{ .Values.storage.compressArtifacts }}"
{{- end }}

{{/*
Define API environment in standalone mode
*/}}
{{- define "testkube-api.env-standalone-mode" -}}
{{- if .Values.mongodb.enabled }}
- name: API_MONGO_DSN
  {{- if .Values.mongodb.secretName }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.mongodb.secretName }}
      key: {{ .Values.mongodb.secretKey }}
  {{- else }}
  value: "{{ .Values.mongodb.dsn }}"
  {{- end }}
{{- if .Values.mongodb.sslCertSecret }}
- name: API_MONGO_SSL_CERT
  value: {{ .Values.mongodb.sslCertSecret }}
  {{- else }}
  {{- if .Values.mongodb.sslCertSecretSecretName }}
- name: API_MONGO_SSL_CERT
  valueFrom:
    secretKeyRef:
      name: {{ .Values.mongodb.sslCertSecretSecretName }}
      key: {{ .Values.mongodb.sslCertSecretSecretKey }}
{{- end }}
{{- end }}
{{- if .Values.mongodb.sslCAFileKey }}
- name: API_MONGO_SSL_CA_FILE_KEY
  value: {{ .Values.mongodb.sslCAFileKey }}
  {{- else }}
  {{- if .Values.mongodb.sslCAFileKeySecretName }}
- name: API_MONGO_SSL_CA_FILE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.mongodb.sslCAFileKeySecretName }}
      key: {{ .Values.mongodb.sslCAFileKeySecretKey }}
  {{- end }}
{{- end }}
{{- if .Values.mongodb.sslClientFileKey }}
- name: API_MONGO_SSL_CLIENT_FILE_KEY
  value: {{ .Values.mongodb.sslClientFileKey }}
  {{- else }}
{{- if .Values.mongodb.sslClientFileKeySecretName }}
- name: API_MONGO_SSL_CLIENT_FILE_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.mongodb.sslClientFileKeySecretName }}
      key: {{ .Values.mongodb.sslClientFileKeySecretKey }}
{{- end }}
{{- end }}
{{- if .Values.mongodb.sslClientFilePassKey }}
- name: API_MONGO_SSL_CLIENT_FILE_PASS_KEY
  value: {{ .Values.mongodb.sslClientFilePassKey }}
  {{- else }}
{{- if .Values.mongodb.sslClientFilePassKeySecretName }}
- name: API_MONGO_SSL_CLIENT_FILE_PASS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ .Values.mongodb.sslClientFilePassKeySecretName }}
      key: {{ .Values.mongodb.sslClientFilePassKeySecretKey }}
  {{- end }}
{{- end }}
{{- if .Values.mongodb.dbType }}
- name: API_MONGO_DB_TYPE
  value: {{ .Values.mongodb.dbType }}
{{- end }}
{{- if .Values.mongodb.allowTLS }}
- name: API_MONGO_ALLOW_TLS
  value: "{{ .Values.mongodb.allowTLS }}"
{{- end }}
- name: API_MONGO_ALLOW_DISK_USE
  value: "{{ .Values.mongodb.allowDiskUse }}"
{{- end }}
{{- if .Values.postgresql.enabled }}
- name: API_POSTGRES_DSN
  value: "{{ .Values.postgresql.dsn }}"
{{- end }}
- name: "NATS_EMBEDDED"
  value: "{{ .Values.nats.embedded }}"
- name: NATS_URI
  {{- if .Values.nats.uri }}
  value: {{ .Values.nats.uri }}
  {{- else if .Values.nats.secretName }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.nats.secretName }}
      key: {{ .Values.nats.secretKey }}
  {{- else }}
  value: "nats://{{ .Release.Name }}-nats"
  {{- end }}
- name: "NATS_SECURE"
  value: "{{ .Values.nats.tls.enabled }}"
{{- if .Values.nats.tls.certSecret.enabled }}
- name: "NATS_CERT_FILE"
  value:  "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.certFile }}"
- name: "NATS_KEY_FILE"
  value: "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.keyFile }}"
{{- if .Values.nats.tls.mountCACertificate }}
- name: "NATS_CA_FILE"
  value: "{{ .Values.nats.tls.certSecret.baseMountPath }}/{{ .Values.nats.tls.certSecret.caFile }}"
{{- end }}
{{- end }}
- name: "STORAGE_ENDPOINT"
  {{- if .Values.storage.endpoint }}
  value:  "{{ .Values.storage.endpoint }}"
  {{- else if .Values.executionNamespaces }}
  value:  "testkube-minio-service-{{ .Release.Namespace }}.{{ .Release.Namespace }}.svc.cluster.local:{{ .Values.storage.endpoint_port }}"
  {{- else }}
  value:  "testkube-minio-service-{{ .Release.Namespace }}:{{ .Values.storage.endpoint_port }}"
  {{- end }}
- name: "STORAGE_BUCKET"
  value:  "{{ .Values.storage.bucket }}"
- name: "STORAGE_EXPIRATION"
  value:  "{{ .Values.storage.expiration }}"
- name: "STORAGE_ACCESSKEYID"
  {{- if .Values.storage.secretNameAccessKeyId }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.storage.secretNameAccessKeyId }}
      key: {{ .Values.storage.secretKeyAccessKeyId }}
  {{- else }}
  value: "{{ .Values.storage.accessKeyId }}"
{{- end }}
- name: "STORAGE_SECRETACCESSKEY"
  {{- if .Values.storage.secretNameSecretAccessKey }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.storage.secretNameSecretAccessKey }}
      key: {{ .Values.storage.secretKeySecretAccessKey }}
  {{- else }}
  value: "{{ .Values.storage.accessKey }}"
{{- end }}
- name: "STORAGE_REGION"
  value: "{{ .Values.storage.region }}"
- name: "STORAGE_TOKEN"
  value: "{{ .Values.storage.token }}"
- name: "STORAGE_SSL"
  value:  "{{ .Values.storage.SSL }}"
- name: "STORAGE_SKIP_VERIFY"
  value: "{{ if hasKey .Values.global.tls "skipVerify" }}{{ .Values.global.tls.skipVerify }}{{ else }}{{ .Values.storage.skipVerify }}{{ end }}"
{{- if .Values.storage.certSecret.enabled }}
- name: "STORAGE_CERT_FILE"
  value:  "{{ .Values.storage.certSecret.baseMountPath }}/{{ .Values.storage.certSecret.certFile }}"
- name: "STORAGE_KEY_FILE"
  value: "{{ .Values.storage.certSecret.baseMountPath }}/{{ .Values.storage.certSecret.keyFile }}"
{{- if .Values.storage.mountCACertificate }}
- name: "STORAGE_CA_FILE"
  value: "{{ .Values.storage.certSecret.baseMountPath }}/{{ .Values.storage.certSecret.caFile }}"
{{- end }}
{{- end }}
- name: "SCRAPPERENABLED"
  value:  "{{ .Values.storage.scrapperEnabled }}"
- name: "COMPRESSARTIFACTS"
  value:  "{{ .Values.storage.compressArtifacts }}"
- name: "LOGS_BUCKET"
  value:  "{{ .Values.logs.bucket }}"
- name: "LOGS_STORAGE"
  {{- if .Values.logs.storage }}
  value:  "{{ .Values.logs.storage }}"
  {{- else }}
  value:  "mongo"
  {{- end }}
{{- end }}

{{/*
Define Test Workflows Toolkit Image
*/}}
{{- define "testkube-tw-toolkit.image" -}}
{{- $registryName := .Values.imageTwToolkit.registry -}}
{{- $repositoryName := .Values.imageTwToolkit.repository -}}
{{- $tag := default .Chart.AppVersion (default .Values.image.tag .Values.imageTwToolkit.tag) | toString -}}
{{- $tagSuffix := .Values.imageTwToolkit.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.imageTwToolkit.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.imageTwToolkit.digest | toString -}}
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
{{- end -}}


{{/*
Define Test Workflows Init Image
*/}}
{{- define "testkube-tw-init.image" -}}
{{- $registryName := .Values.imageTwInit.registry -}}
{{- $repositoryName := .Values.imageTwInit.repository -}}
{{- $tag := default .Chart.AppVersion (default .Values.image.tag .Values.imageTwInit.tag) | toString -}}
{{- $tagSuffix := .Values.imageTwInit.tagSuffix -}}
{{- $separator := ":" -}}
{{- if .Values.imageTwInit.digest }}
    {{- $separator = "@" -}}
    {{- $tag = .Values.imageTwInit.digest | toString -}}
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
{{- end -}}

{{/*
Define TESTKUBE_WATCHER_NAMESPACES variable
*/}}
{{- define "testkube-api.watcher-namespaces" -}}
{{- if .Values.multinamespace.enabled }}
{{ join "," (concat (list .Release.Namespace) .Values.additionalNamespaces) }}
{{- else }}
{{- printf "" }}
{{- end }}
{{- end }}

{{/*
Define podSecurityContext
*/}}
{{- define "testkube-api.podSecurityContext" -}}
{{- if .Values.global.podSecurityContext }}
{{ toYaml .Values.global.podSecurityContext }}
{{- else }}
{{ toYaml .Values.podSecurityContext }}
{{- end }}
{{- end }}

{{/*
Define containerSecurityContext
*/}}
{{- define "testkube-api.containerSecurityContext" -}}
{{- if .Values.global.containerSecurityContext }}
{{- toYaml .Values.global.containerSecurityContext }}
{{- else }}
{{- toYaml .Values.securityContext }}
{{- end }}
{{- end }}

{{/*
Define podSecurityContext for MinIo
*/}}
{{- define "minio.podSecurityContext" -}}
{{- if .Values.global.podSecurityContext }}
{{ toYaml .Values.global.podSecurityContext }}
{{- else }}
{{ toYaml .Values.minio.podSecurityContext }}
{{- end }}
{{- end }}

{{/*
Define containerSecurityContext for MinIo
*/}}
{{- define "minio.containerSecurityContext" -}}
{{- if .Values.global.containerSecurityContext }}
{{- toYaml .Values.global.containerSecurityContext }}
{{- else }}
{{- toYaml .Values.minio.securityContext }}
{{- end }}
{{- end }}
