{{/* PVC template for test with artifact requests */}}
{{- define "testkube-api.pvc-template" -}}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{`{{ .Name }}`}}-pvc
  namespace: {{`{{ .Namespace }}`}}
spec:
  {{`{{- if .ArtifactRequest.StorageClassName }}`}}
  storageClassName: {{`{{ .ArtifactRequest.StorageClassName }}`}}
  {{`{{- else if .ArtifactRequest.UseDefaultStorageClassName }}`}}
  storageClassName: {{`{{ .DefaultStorageClassName }}`}}
  {{`{{- end }}`}}
  accessModes:
  {{`{{- if .ArtifactRequest.SharedBetweenPods }}`}}
    - ReadWriteMany
  {{`{{- else }}`}}
    - ReadWriteOnce
  {{`{{- end }}`}}
  resources:
    requests:
      storage: {{ .Values.storageRequest | quote }}
{{- end }}
