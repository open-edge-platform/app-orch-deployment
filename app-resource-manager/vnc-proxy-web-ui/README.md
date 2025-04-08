<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# VNC UI

This is a VNC UI that allows access to a VM running on an Edge Node.

It includes library files:

- `keycloak.min.js` - retrieved from [keycloak.min.js]
- `rfb.js` - using the NoVNC library `make rollup-rfb` to create a single file

## Usage

The web page redirects immediately to Keycloak for login. After login, the user is redirected
back to the web page with a token, since it uses the Keycloak client library.

The JavaScript first checks that the expected query parameters are present and forms them into
a WebSocket URL.

The expected query parameters are:

- project - the project ID
- app - the app ID
- cluster - the cluster ID
- vm - the VM ID

An example URL is:
`https://vnc.kind.internal/?project=26abfd98-d59e-4daf-a164-62535f8bb92f&app=b-68383349-afd9-506f-b76c-764dae2a05fd&cluster=cluster-9bda5c32&vm=5f76485f-114e-4344-90ad-af4448c556a9`

It then extracts the token from the Keycloak client, breaks it into chunks of 2k, and saves it as
one or more cookies.

Finally, the NoVNC library's RFB object is created in the UI and given the WebSocket URL. It calls
out to this URL and the browser sends the Keycloak cookies with the request. The request is handled
by the VNC proxy, which checks the token and establishes a WebSocket tunnel to KubeVirt on the
Edge Node (the correct one is selected by path parameters given in the WebSocket URL).

## Development

In production, ARM serves up this HTML and JavaScript, but you can run it locally for testing using

```shell
python3 -m http.server 3000
```

And then open a browser to
`http://localhost:3000/?project=26abfd98-d59e-4daf-a164-62535f8bb92f&app=b-68383349-afd9-506f-b76c-764dae2a05fd&cluster=cluster-9bda5c32&vm=5f76485f-114e-4344-90ad-af4448c556a9`

If any of the query parameters are missing, the page will display an error message.

[keycloak.min.js]: https://cdn.jsdelivr.net/npm/keycloak-js@25.0.5/dist/keycloak.min.js
