{{- /*
SPDX-FileCopyrightText: (C) 2023 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/ -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "app-deployment-manager.name" -}}
{{- default .Chart.Name .Values.adm.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "app-deployment-manager.fullname" -}}
{{- if .Values.adm.fullnameOverride -}}
{{- .Values.adm.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.adm.nameOverride -}}
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
{{- define "app-deployment-manager.labels" -}}
app: {{ template "app-deployment-manager.name" . }}
app.kubernetes.io/name: {{ template "app-deployment-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: "controller"
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "app-deployment-manager.serviceAccountName" -}}
{{- if .Values.adm.serviceAccount.create -}}
    {{ default (include "app-deployment-manager.fullname" .) .Values.adm.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.adm.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
app deployment manager api.
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "app-deployment-manager-api.name" -}}
{{- default .Chart.Name .Values.gateway.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Renders a set of standardised labels for app deployment manager api resources.
*/}}
{{- define "app-deployment-manager-api.labels" -}}
app: {{ template "app-deployment-manager-api.name" . }}
app.kubernetes.io/name: {{ template "app-deployment-manager-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: "api"
{{- end -}}

{{/*
Renders a set of standardised selector labels for app deployment manager api resources.
*/}}
{{- define "app-deployment-manager-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app-deployment-manager-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}