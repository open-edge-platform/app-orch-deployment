{{/*
# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "interconnect-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "interconnect-manager.fullname" -}}
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
{{- define "interconnect-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "interconnect-manager.labels" -}}
helm.sh/chart: {{ include "interconnect-manager.chart" . }}
{{ include "interconnect-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "interconnect-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "interconnect-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app: {{ template "interconnect-manager.name" . }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "interconnect-manager.serviceAccountName" -}}
{{- if .Values.interconnect_manager.serviceAccount.create }}
{{- default (include "interconnect-manager.fullname" .) .Values.interconnect_manager.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.interconnect_manager.serviceAccount.name }}
{{- end }}
{{- end }}
