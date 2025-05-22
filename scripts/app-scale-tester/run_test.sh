#!/bin/bash

clusterFqdn=integration14.espd.infra-host.com
appOrchEndpoint="https://api.${clusterFqdn}"
enspUser=intel-itep-user
enspPassword=ChangeMeOn1stLogin!
project=itep
orchAPIToken=$(curl -s --location --request POST https://keycloak.${clusterFqdn}/realms/master/protocol/openid-connect/token \
      -H 'Content-Type: application/x-www-form-urlencoded' \
      -d "username=${enspUser}" \
      -d "password=${enspPassword}" \
      -d "grant_type=password" \
      -d "client_id=system-client" \
      -d "scope=openid" |
      jq -r '.access_token' 2>/dev/null)

