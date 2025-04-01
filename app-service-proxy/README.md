# Application Service Proxy

Application Service Proxy is a HTTP/HTTPs reverse proxy for application services running on edge clusters. Along with the Application Service Agent running on edge cluster, it provides external access to web applications via the central clusters. It also provides basic authentication and authorization based on the HTTP method type.

## Websocket connection setup

Application Service Agent runs on the cluster where the application, whose API is to be proxied, is running. It opens a WebSocket based tunnel to Application Service Proxy to relay received API calls to application API service. Since it is the agent that initiates the WebSocket connection, edges can remain behind the FW.


## User request

![User request](docs/api-proxy-user-request.drawio.png)

1. User makes a request to *https://app-service-proxy.kind.internal/clusters/{cluster-id}/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}* to access a *service* running on an edge cluster. Optionally, the scheme can also be provided as *{scheme:}/{service}:{port}*, where *scheme* is either https or http (default).

2. Application Service Proxy forwards the request to the Service Agent over the websocket tunnel identified by *cluster-id*. Service Proxy strips the prefix /clusters/{*cluster-id*} from the requested url. The requested url is now *https://app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}*.

3. The Service Agent simply forwards the request. As a prerequisite, there is a DNS alias in the edge cluster that maps *app-service-proxy.kind.internal* to *kubernetes.default.svc.cluster.local*. This results in the request *https://app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}* being forwarded to the Kubernetes Api-Server.

4. The Kube Api-Server forwards the request to the app's service as *http://{service}:{port}/{path}*.

5. The Kubernetes API-Server rewrites urls in the body of the response replacing *{service}.{namespace}:{port}/{path}* with *app-service-proxy.kind.internal/api/v1/namespaces/{namespace}/services/{service}:{port}/proxy/{path}*.

6. The API Server rewrites urls in the body of the response by prefixing the cluster info, */clusters/{cluster-id}*
