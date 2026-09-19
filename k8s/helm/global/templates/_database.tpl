{{/*
Database selection preservation helpers.

These live in the `global` chart because it is a dependency of BOTH the umbrella
`testkube` chart (which creates the marker ConfigMap and the preflight guard) and
the leaf `testkube-api` chart (which reads the effective database to wire the API
DSN). Defining them here guarantees both scopes resolve the same value and the
same marker name.

All lookups below are read-only. `lookup` returns empty during `helm template`,
`helm lint`, client-side dry-run and Argo CD renders, so in those cases resolution
falls through to the sentinel / enabled-flags fallback.

Note: `dig` is intentionally avoided — it type-asserts its argument to
map[string]interface{}, which fails on Helm's named `chartutil.Values` type. Safe
field access with `default dict` works on both plain maps and Values.
*/}}

{{/*
Marker ConfigMap name used to persist the selected database across upgrades.
Reads the optional override from the shared `global` scope so the umbrella (which
creates it) and the leaf (which reads it) always agree, regardless of scope.
Usage: {{ include "testkube.databaseMarkerName" . }}
*/}}
{{- define "testkube.databaseMarkerName" -}}
{{- $global := .Values.global | default dict -}}
{{- $dbCfg := $global.database | default dict -}}
{{- $override := $dbCfg.markerName | default "" -}}
{{- default (printf "%s-db-marker" .Release.Name) $override -}}
{{- end -}}

{{/*
Resolve the effective database for this release: "mongodb" or "postgresql".

Priority:
  1. Explicit sentinel .Values.global.database.type ("mongodb" | "postgresql").
     The empty default lets us distinguish a real user choice from a chart
     default, and it is the only signal that works where `lookup` is empty
     (helm template / Argo CD).
  2. Persisted marker ConfigMap written by a previous release (see marker template).
  3. Live detection: bundled engine workloads by fixed name, then the API server
     Deployment's DSN env vars (mirrors the Go server selection: postgres wins).
  4. Fallback to the enabled flags (postgres wins if both) — this reproduces the
     historical behaviour and covers fresh installs and lookup-empty renders.

Usage: {{ include "testkube.effectiveDatabase" . | trim }}
*/}}
{{- define "testkube.effectiveDatabase" -}}
{{- $ns := .Release.Namespace -}}
{{- $global := .Values.global | default dict -}}
{{- $dbCfg := $global.database | default dict -}}
{{- $result := "" -}}
{{/* 1. explicit sentinel */}}
{{- $sentinel := $dbCfg.type | default "" -}}
{{- if or (eq $sentinel "mongodb") (eq $sentinel "postgresql") -}}
{{- $result = $sentinel -}}
{{- end -}}
{{/* 2. persisted marker */}}
{{- if not $result -}}
{{- $marker := lookup "v1" "ConfigMap" $ns (include "testkube.databaseMarkerName" .) -}}
{{- if and $marker $marker.data -}}
{{- $stored := trim (default "" (get $marker.data "database")) -}}
{{- if or (eq $stored "mongodb") (eq $stored "postgresql") -}}
{{- $result = $stored -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{/* 3a. live detection via engine workloads */}}
{{- if not $result -}}
{{- if or (lookup "apps/v1" "StatefulSet" $ns "testkube-postgresql") (lookup "apps/v1" "Deployment" $ns "testkube-postgresql") -}}
{{- $result = "postgresql" -}}
{{- else if or (lookup "apps/v1" "StatefulSet" $ns "testkube-mongodb") (lookup "apps/v1" "Deployment" $ns "testkube-mongodb") -}}
{{- $result = "mongodb" -}}
{{- end -}}
{{- end -}}
{{/* 3b. live detection via api-server DSN env (postgres wins) */}}
{{- if not $result -}}
{{- $apiName := $global.apiFullname | default "testkube-api-server" -}}
{{- $dep := lookup "apps/v1" "Deployment" $ns $apiName -}}
{{- if $dep -}}
{{- $hasPg := false -}}
{{- $hasMongo := false -}}
{{- range $c := $dep.spec.template.spec.containers -}}
{{- range $e := $c.env -}}
{{- if eq $e.name "API_POSTGRES_DSN" -}}{{- $hasPg = true -}}{{- end -}}
{{- if eq $e.name "API_MONGO_DSN" -}}{{- $hasMongo = true -}}{{- end -}}
{{- end -}}
{{- end -}}
{{- if $hasPg -}}
{{- $result = "postgresql" -}}
{{- else if $hasMongo -}}
{{- $result = "mongodb" -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{/* 4. fallback to enabled flags (postgres wins), defaulting to mongodb */}}
{{- if not $result -}}
{{- $pg := .Values.postgresql | default dict -}}
{{- $mongo := .Values.mongodb | default dict -}}
{{- if $pg.enabled -}}
{{- $result = "postgresql" -}}
{{- else if $mongo.enabled -}}
{{- $result = "mongodb" -}}
{{- else -}}
{{- $result = "mongodb" -}}
{{- end -}}
{{- end -}}
{{- $result -}}
{{- end -}}
