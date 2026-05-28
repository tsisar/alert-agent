{{/*
Common labels
*/}}
{{- define "alert-agent.labels" -}}
app.kubernetes.io/name: alert-agent
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "alert-agent.selectorLabels" -}}
app.kubernetes.io/name: alert-agent
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Check if any secret value is non-empty
*/}}
{{- define "alert-agent.hasSecrets" -}}
{{- $has := false -}}
{{- range $key, $value := .Values.secret -}}
  {{- if $value -}}
    {{- $has = true -}}
  {{- end -}}
{{- end -}}
{{- if $has -}}true{{- end -}}
{{- end }}