{{- /*
SPDX-FileCopyrightText: (C) 2023 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/ -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "app-service-proxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "app-service-proxy.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Renders a set of standardised labels for app deployment manager resources.
*/}}
{{- define "app-service-proxy.labels" -}}
app: {{ template "app-service-proxy.name" . }}
app.kubernetes.io/name: {{ template "app-service-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "app-service-proxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app-service-proxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "app-service-proxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "app-service-proxy.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}
