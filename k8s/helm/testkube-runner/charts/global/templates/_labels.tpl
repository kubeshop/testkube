{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "global.version.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Kubernetes standard labels
*/}}
{{- define "global.labels.standard" -}}
helm.sh/chart: {{ include "global.version.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

