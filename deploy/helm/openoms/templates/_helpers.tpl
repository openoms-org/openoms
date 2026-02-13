{{/*
Expand the name of the chart.
*/}}
{{- define "openoms.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "openoms.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "openoms.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels shared across all resources.
*/}}
{{- define "openoms.labels" -}}
helm.sh/chart: {{ include "openoms.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: openoms
{{- end }}

{{/*
Selector labels for api-server.
*/}}
{{- define "openoms.apiServer.selectorLabels" -}}
app.kubernetes.io/name: openoms-api
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels for worker.
*/}}
{{- define "openoms.worker.selectorLabels" -}}
app.kubernetes.io/name: openoms-worker
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels for dashboard.
*/}}
{{- define "openoms.dashboard.selectorLabels" -}}
app.kubernetes.io/name: openoms-dashboard
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
API server container image.
*/}}
{{- define "openoms.apiServer.image" -}}
{{- $tag := default .Chart.AppVersion .Values.apiServer.image.tag -}}
{{- printf "%s:%s" .Values.apiServer.image.repository $tag }}
{{- end }}

{{/*
Dashboard container image.
*/}}
{{- define "openoms.dashboard.image" -}}
{{- $tag := default .Chart.AppVersion .Values.dashboard.image.tag -}}
{{- printf "%s:%s" .Values.dashboard.image.repository $tag }}
{{- end }}

{{/*
Migration container image.
*/}}
{{- define "openoms.migration.image" -}}
{{- $tag := default .Chart.AppVersion .Values.migration.image.tag -}}
{{- printf "%s:%s" .Values.migration.image.repository $tag }}
{{- end }}

{{/*
Secret name to use.
*/}}
{{- define "openoms.secretName" -}}
{{- .Values.secrets.name }}
{{- end }}
