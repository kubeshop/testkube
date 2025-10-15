{{/* Pod template for slave pods */}}
{{- define "testkube-api.slave-pod-template" -}}
apiVersion: v1
kind: Pod
metadata:
  name: "{{`{{ .Name }}`}}"
  namespace: {{`{{ .Namespace }}`}}
  ownerReferences:
  - apiVersion: batch/v1
    kind: job
    name: {{`{{ .JobName }}`}}
    uid: {{`{{ .JobUID }}`}}
  {{- with .Values.jobPodAnnotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{`{{- if gt .ActiveDeadlineSeconds 0 }}`}}
  activeDeadlineSeconds: {{`{{ .ActiveDeadlineSeconds }}`}}
  {{`{{- end }}`}}
  {{`{{- if not (and  .ArtifactRequest (eq .ArtifactRequest.VolumeMountPath "/data")) }}`}}
  initContainers:
  - name: init
    {{`{{- if .Registry }}`}}
    image: {{`{{ .Registry }}`}}/{{`{{ .InitImage }}`}}
    {{`{{- else }}`}}
    image: {{`{{ .InitImage }}`}}
    {{`{{- end }}`}}
    imagePullPolicy: IfNotPresent
    command:
      - "/bin/runner"
      - '{{`{{ .Jsn }}`}}'
    {{- with .Values.initContainerResources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{`{{- if .RunnerCustomCASecret }}`}}
    env:
      - name: SSL_CERT_DIR
        value: /etc/testkube/certs
      - name: GIT_SSL_CAPATH
        value: /etc/testkube/certs
    {{`{{- end }}`}}
    volumeMounts:
    - name: data-volume
      mountPath: /data
    {{`{{- if .CertificateSecret }}`}}
    - name: {{`{{ .CertificateSecret }}`}}
      mountPath: /etc/certs
    {{`{{- end }}`}}
    {{`{{- if .RunnerCustomCASecret }}`}}
    - name: {{`{{ .RunnerCustomCASecret }}`}}
      mountPath: /etc/testkube/certs/testkube-custom-ca.pem
      readOnly: true
      subPath: {{ .Values.cloud.tls.customCaSecretKey }}
    {{`{{- end }}`}}
    {{`{{- if .ArtifactRequest }}`}}
      {{`{{- if and .ArtifactRequest.VolumeMountPath (or .ArtifactRequest.StorageClassName .ArtifactRequest.UseDefaultStorageClassName) }}`}}
    - name: artifact-volume
      mountPath: {{`{{ .ArtifactRequest.VolumeMountPath }}`}}
      {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{`{{- range $configmap := .EnvConfigMaps }}`}}
    {{`{{- if and $configmap.Mount $configmap.Reference }}`}}
    - name: {{`{{ $configmap.Reference.Name }}`}}
      mountPath: {{`{{ $configmap.MountPath }}`}}
    {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{`{{- range $secret := .EnvSecrets }}`}}
    {{`{{- if and $secret.Mount $secret.Reference }}`}}
    - name: {{`{{ $secret.Reference.Name }}`}}
      mountPath: {{`{{ $secret.MountPath }}`}}
    {{`{{- end }}`}}
    {{`{{- end }}`}}
  {{`{{ end }}`}}
  containers:
  {{`{{ if .Features.LogsV2 -}}`}}
  - name: "main-logs"
    {{`{{- if .Registry }}`}}
    image: {{`{{ .Registry }}`}}/{{`{{ .LogSidecarImage }}`}}
    {{`{{- else }}`}}
    image: {{`{{ .LogSidecarImage }}`}}
    {{`{{- end }}`}}
    imagePullPolicy: IfNotPresent
    {{- with .Values.logsV2ContainerResources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: NAMESPACE
      value: {{`{{ .Namespace }}`}}
    - name: NATS_URI
      value: {{`{{ .NatsUri }}`}}
    - name: ID
      value: {{`{{ .JobName }}`}}
    - name: GROUP
      value: test-slave
    - name: SOURCE
      value: "job-slave-pod:{{`{{ .Name }}`}}"
  {{`{{- end }}`}}
  - name: main
    {{`{{- if .Registry }}`}}
    image: {{`{{ .Registry }}`}}/{{`{{ .Image }}`}}
    {{`{{- else }}`}}
    image: {{`{{ .Image }}`}}
    {{`{{- end }}`}}
    imagePullPolicy: IfNotPresent
    {{- with .Values.containerResources }}
    resources:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    env:
    {{- if .Values.global.tls.caCertPath }}
    - name: SSL_CERT_DIR
      value: {{ .Values.global.tls.caCertPath }}
    - name: GIT_SSL_CAPATH
      value: {{ .Values.global.tls.caCertPath }}
    {{- end }}
    ports:
    {{`{{- range $port := .Ports }}`}}
    - name: {{`{{ $port.Name }}`}}
      containerPort: {{`{{ $port.ContainerPort }}`}}
    {{`{{- end}}`}}
    {{`{{- range $port := .Ports }}`}}
    {{`{{- if eq $port.Name "server-port" }}`}}
    livenessProbe:
      tcpSocket:
        port: {{`{{ $port.ContainerPort }}`}}
      failureThreshold: 3
      periodSeconds: 5
      successThreshold: 1
      timeoutSeconds: 1
    readinessProbe:
      tcpSocket:
        port: {{`{{ $port.ContainerPort }}`}}
      failureThreshold: 3
      initialDelaySeconds: 10
      periodSeconds: 5
      timeoutSeconds: 1
    {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{`{{- if .Resources }}`}}
    resources:
      {{`{{- if .Resources.Limits }}`}}
      limits:
        {{`{{- if .Resources.Limits.Cpu }}`}}
        cpu: {{`{{ .Resources.Limits.Cpu }}`}}
        {{`{{- end }}`}}
        {{`{{- if .Resources.Limits.Memory }}`}}
        memory: {{`{{ .Resources.Limits.Memory }}`}}
        {{`{{- end }}`}}
      {{`{{- end }}`}}
      {{`{{- if .Resources.Requests }}`}}
      requests:
        {{`{{- if .Resources.Requests.Cpu }}`}}
        cpu: {{`{{ .Resources.Requests.Cpu }}`}}
        {{`{{- end }}`}}
        {{`{{- if .Resources.Requests.Memory }}`}}
        memory: {{`{{ .Resources.Requests.Memory }}`}}
        {{`{{- end }}`}}
      {{`{{- end }}`}}
    {{`{{- end }}`}}
    volumeMounts:
    {{`{{- if not (and  .ArtifactRequest (eq .ArtifactRequest.VolumeMountPath "/data")) }}`}}
    - name: data-volume
      mountPath: /data
    {{`{{ end }}`}}
    {{`{{- if .CertificateSecret }}`}}
    - name: {{`{{ .CertificateSecret }}`}}
      mountPath: /etc/certs
    {{`{{- end }}`}}
    {{`{{- if .ArtifactRequest }}`}}
      {{`{{- if and .ArtifactRequest.VolumeMountPath (or .ArtifactRequest.StorageClassName .ArtifactRequest.UseDefaultStorageClassName) }}`}}
    - name: artifact-volume
      mountPath: {{`{{ .ArtifactRequest.VolumeMountPath }}`}}
      {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{`{{- range $configmap := .EnvConfigMaps }}`}}
    {{`{{- if and $configmap.Mount $configmap.Reference }}`}}
    - name: {{`{{ $configmap.Reference.Name }}`}}
      mountPath: {{`{{ $configmap.MountPath }}`}}
    {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{`{{- range $secret := .EnvSecrets }}`}}
    {{`{{- if and $secret.Mount $secret.Reference }}`}}
    - name: {{`{{ $secret.Reference.Name }}`}}
      mountPath: {{`{{ $secret.MountPath }}`}}
    {{`{{- end }}`}}
    {{`{{- end }}`}}
    {{- with .Values.additionalJobVolumeMounts }}
    {{- toYaml . | nindent 4 -}}
    {{- end }}
    {{- with .Values.global.volumes.additionalVolumeMounts }}
    {{- toYaml . | nindent 4 -}}
    {{- end }}
  volumes:
  {{`{{- if not (and  .ArtifactRequest (eq .ArtifactRequest.VolumeMountPath "/data")) }}`}}
  - name: data-volume
    emptyDir: {}
  {{`{{ end }}`}}
  {{`{{- if .CertificateSecret }}`}}
  - name: {{`{{ .CertificateSecret }}`}}
    secret:
      secretName: {{`{{ .CertificateSecret }}`}}
  {{`{{- end }}`}}
  {{`{{- if .RunnerCustomCASecret }}`}}
  - name: {{`{{ .RunnerCustomCASecret }}`}}
    secret:
      secretName: {{`{{ .RunnerCustomCASecret }}`}}
      defaultMode: 420
  {{`{{- end }}`}}
  {{`{{- if .ArtifactRequest }}`}}
    {{`{{- if and .ArtifactRequest.VolumeMountPath (or .ArtifactRequest.StorageClassName .ArtifactRequest.UseDefaultStorageClassName) }}`}}
  - name: artifact-volume
    persistentVolumeClaim:
      claimName: {{`{{ .JobName }}`}}-pvc
    {{`{{- end }}`}}
  {{`{{- end }}`}}
  {{`{{- range $configmap := .EnvConfigMaps }}`}}
  {{`{{- if and $configmap.Mount $configmap.Reference }}`}}
  - name: {{`{{ $configmap.Reference.Name }}`}}
    configmap:
      name: {{`{{ $configmap.Reference.Name }}`}}
  {{`{{- end }}`}}
  {{`{{- end }}`}}
  {{`{{- range $secret := .EnvSecrets }}`}}
  {{`{{- if and $secret.Mount $secret.Reference }}`}}
  - name: {{`{{ $secret.Reference.Name }}`}}
    secret:
      secretName: {{`{{ $secret.Reference.Name }}`}}
  {{`{{- end }}`}}
  {{`{{- end }}`}}
  {{- with .Values.additionalJobVolumes }}
  {{- toYaml . | nindent 2 -}}
  {{- end }}
  {{- with .Values.global.volumes.additionalVolumes }}
  {{- toYaml . | nindent 2 -}}
  {{- end }}
  restartPolicy: Always
  {{`{{- if .ServiceAccountName }}`}}
  serviceAccountName: {{`{{ .ServiceAccountName }}`}}
  {{`{{- end }}`}}
  {{- with (default .Values.imagePullSecrets .Values.global.imagePullSecrets) }}
  imagePullSecrets:
    {{- range . }}
    {{- if typeIsLike "map[string]interface {}" . }}
  - name: {{ .name | quote }}
    {{- else }}
  - name: {{ . | quote  }}
    {{- end }}
    {{- end }}
    {{- end }}
{{- end }}
