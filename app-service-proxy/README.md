<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Application Service Proxy

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Application Service Proxy is an HTTP/HTTPS reverse proxy for application services running on edge clusters.
Along with the Application Service Agent running on the edge cluster, it provides external access to web applications
via the central clusters. It also provides basic authentication and authorization based on the HTTP method
type.

## WebSocket Connection Setup

The Application Service Agent runs on the cluster where the application, whose API is to be proxied, is running.
It opens a WebSocket-based tunnel to the Application Service Proxy to relay received API calls to the application
API service. Since it is the agent that initiates the WebSocket connection, edges can remain behind the firewall.

## User Request

1. The user makes a request to `https://app-service-proxy.kind.internal/clusters/{cluster-id}/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}`
   to access a service running on an edge cluster. Optionally, the scheme can also be provided as
   `{scheme:}/{service}:{port}`where `scheme` is either https or http (default).

2. The Application Service Proxy forwards the request to the Service Agent over the WebSocket tunnel identified by
   `cluster-id`. The Service Proxy strips the prefix `/clusters/{cluster-id}` from the requested URL.
   The requested URL is now `https://app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}`.

3. The Service Agent simply forwards the request. As a prerequisite, there is a DNS alias in the edge cluster that maps
   `app-service-proxy.kind.internal` to `kubernetes.default.svc.cluster.local`. This results in the request
   `https://app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}`
   being forwarded to the Kubernetes API Server.

4. The Kubernetes API Server forwards the request to the app's service as `http://{service}:{port}/{path}`.

5. The Kubernetes API Server rewrites URLs in the body of the response, replacing `{service}.{namespace}:{port}/{path}`
   with `app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}`.

6. The API Server rewrites URLs in the body of the response by prefixing the cluster info, `/clusters/{cluster-id}`.
