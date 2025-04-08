<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# App Service Proxy Web Login JavaScript

This is a simple web page and JavaScript that can be used to log in to the App Service Proxy.

## Usage

The web page is a simple program that redirects immediately to Keycloak for login. After login, the user is redirected
back to the web page with a token, since it uses the Keycloak client library.

The JavaScript first checks that the expected query parameters are present and saves each as a cookie.

It then extracts the token from the Keycloak client, breaks it into chunks of 2k, and saves it
as cookies. The cookies can then be read by the App Service Proxy and used to authenticate the user.

In production, the ASP serves up this HTML and JavaScript, but you can run it locally for testing using

```shell
python3 -m http.server 3000
```

And then open a browser to `http://localhost:3000/app-service-proxy-index.html?project=p2&cluster=c2&namespace=n2&service=s2&port=1234`

If any of the query parameters are missing, the page will display an error message.
