{{/*
Return the proper image name
{{ include "global.images.image" ( dict "imageRoot" .Values.path.to.the.image "global" $) }}
*/}}
{{- define "global.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $separator := ":" -}}
{{- $termination := .imageRoot.tag   | toString -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
     {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if .imageRoot.digest }}
    {{- $separator = "@" -}}
    {{- $termination = .imageRoot.digest | toString -}}
{{- end -}}
{{- printf "%s/%s%s%s" $registryName $repositoryName $separator $termination -}}
{{- end -}}


{{/*
Return the proper Docker Image Registry Secret Names evaluating values as templates
{{ include "global.images.renderPullSecrets" . }}
*/}}
{{- define "global.images.renderPullSecrets" -}}
{{- $context := . }}
{{- $global := index $context "global" }}
{{- $path := index $context "secretPath" }}
{{- if $global.imagePullSecrets }}
imagePullSecrets:
{{- range $global.imagePullSecrets }}
{{- if typeIsLike "map[string]interface {}" . }}
- name: {{ .name | quote }}
{{- else }}
- name: {{ . | quote  }}
{{- end }}
{{- end }}
{{- else -}}
{{- if $path }}
imagePullSecrets:
{{- range $path }}
- name: {{ . }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
