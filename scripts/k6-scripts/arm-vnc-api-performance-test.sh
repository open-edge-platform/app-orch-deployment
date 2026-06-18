#!/bin/bash

set -eu

# Initialize some global variables and constants
clusterFqdn=kind.internal # Change if needed
enspUser=all-groups-example-user && enspPassword=ChangeMeOn1stLogin!
appID=$2
appEndpointsPerUser=1
users=$1

currentTimeStamp=$(date -u +"%Y%m%d-%H%M%S")
resultsDirectory="./"
tmpJSONFile="$resultsDirectory/$currentTimeStamp.json"

# Function to initialize Orchestrator API Token
initializeOrchAPIToken() {
  orchAPIToken=$(curl -kX POST https://keycloak.${clusterFqdn}/realms/master/protocol/openid-connect/token \
    -d "username=${enspUser}" \
    -d "password=${enspPassword}" \
    -d "grant_type=password" \
    -d "client_id=system-client" \
    -d "scope=openid" |
    jq -r '.access_token' 2>/dev/null)
}

runApiLatencyChecks() {

  echo "Running API latency tests... it will take a few mins to capture the results"
  initializeOrchAPIToken
  k6 run --env MY_HOSTNAME="$clusterFqdn" --env API_TOKEN="$orchAPIToken" --env APP_ID="$appID" --env USERS="$users" --env APPS_PER_USER="$appEndpointsPerUser" ./arm-vnc-api-latency.js -q --no-thresholds --summary-export="$tmpJSONFile"
}

runApiLatencyChecks

#wscat --no-check --header "origin: https://web-ui.kind.internal" --header "cookie: keycloak-token=$orchAPIToken" -c wss://vnc.kind.internal/vnc/b-867bcd29-f8ba-5b12-b114-e6b21553cb2d/cluster-01234567/c7d7a6b3-dccb-4402-b2f7-4e0cd7a85c41