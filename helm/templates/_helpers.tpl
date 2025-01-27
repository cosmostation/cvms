{{/*
Expand the name of the chart.
*/}}
{{- define "cvms.name" -}}
{{- coalesce .Release.Name .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cvms.fullname" -}}
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
{{- define "cvms.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cvms.labels" -}}
helm.sh/chart: {{ include "cvms.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Metrics labels for indexer
*/}}
{{- define "cvms.indexerSelectorMetricLabels" -}}
app.kubernetes.io/metrics-name: {{ include "cvms.name" . }}-indexer
app.kubernetes.io/metrics-instance: {{ .Release.Name }}-indexer
{{- end }}

{{/*
Metrics labels for exporter
*/}}
{{- define "cvms.exporterSelectorMetricLabels" -}}
app.kubernetes.io/metrics-name: {{ include "cvms.name" . }}-exporter
app.kubernetes.io/metrics-instance: {{ .Release.Name }}-exporter
{{- end }}

{{- if and (not .Values.postgresql.enabled) (not .Values.postgresql.external.host) -}}
{{- fail "NO DATABASE: You disabled the PostgreSQL sub-chart but did not specify an external PostgreSQL host. Either deploy the PostgreSQL sub-chart by setting postgresql.enabled=true or set a value for postgresql.external.host." -}}
{{- end }}

{{- if and .Values.postgresql.enabled .Values.postgresql.external.host -}}
{{- fail "CONFLICT: You specified to deploy the PostgreSQL sub-chart and also specified an external PostgreSQL instance. Only one of postgresql.enabled (deploy sub-chart) and postgresql.external.host can be set." -}}
{{- end }}

{{- if .Values.customChainsConfig.enabled -}}
{{- if not .Values.customChainsConfig.name -}}
{{- fail "A name is required for customChainsConfig when enabled" -}}
{{- end }}
{{- end }}

{{- if not .Values.cvmsConfig.name }}
{{- fail "A name is required for cvmsConfig" }}
{{- end }}

{{/*
Fail the Helm chart if monikers is an empty list
*/}}
{{- define "validateMonikers" -}}
{{- if eq (len .Values.cvmsConfig.monikers) 0 }}
{{- fail "The 'monikers' list in 'cvmsConfig' must not be empty. Please provide at least one moniker." }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cvms.postgresql.fullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" "postgresql" "chartValues" .Values.postgresql "context" $) -}}
{{- end -}}

{{/*
Get PostgreSQL host
*/}}
{{- define "cvms.postgresql.host" -}}
{{- ternary (include "cvms.postgresql.fullname" .) .Values.postgresql.external.host .Values.postgresql.enabled | quote -}}
{{- end -}}

{{/*
Get PostgreSQL port
*/}}
{{- define "cvms.postgresql.port" -}}
{{- if .Values.postgresql.enabled -}}
    {{- default "5432"  .Values.postgresql.port | quote -}}
{{- else -}}
    {{- default "5432"  .Values.postgresql.external.port | quote -}}
{{- end -}}
{{- end -}}

{{/*
Get PostgreSQL user
*/}}
{{- define "cvms.postgresql.user" -}}
{{- ternary .Values.postgresql.auth.username .Values.postgresql.external.user .Values.postgresql.enabled | quote -}}
{{- end -}}

{{/*
Get PostgreSQL password
*/}}
{{- define "cvms.postgresql.password" -}}
{{- ternary .Values.postgresql.auth.password .Values.postgresql.external.password .Values.postgresql.enabled | quote -}}
{{- end -}}

{{/*
Get PostgreSQL database
*/}}
{{- define "cvms.postgresql.database" -}}
{{- ternary .Values.postgresql.auth.database .Values.postgresql.external.database .Values.postgresql.enabled | quote -}}
{{- end -}}