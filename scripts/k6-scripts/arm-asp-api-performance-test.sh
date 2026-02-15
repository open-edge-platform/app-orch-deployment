#!/bin/bash

set -eu

# Initialize some global variables and constants
clusterFqdn=scale.espd.infra-host.com # Change if needed
enspUser=intel-itep-user && enspPassword=ChangeMeOn1stLogin!
prometheusURL="https://observability-admin.${clusterFqdn}/api/datasources/uid/orchestrator-mimir/resources/api/v1"
appID=""
observabilityApiCredentials=""
appEndpointsPerUser=100

currentTimeStamp=$(date -u +"%Y%m%d-%H%M%S")
resultsDirectory="./test-results/$currentTimeStamp"
tmpJSONFile="$resultsDirectory/tmp.json"

# CSV Files to capture the latency and Resource metrics
armAvgAPILatencyCsv="arm-api-latency.csv"
aspAvgAPILatencyCsv="asp-api-latency.csv"
avgCpuOrchAppCsv="avg-cpu-usage-ma.csv"
maxCpuOrchAppCsv="max-cpu-usage-ma.csv"
avgRamOrchAppCsv="avg-ram-usage-ma.csv"
maxRamOrchAppCsv="max-ram-usage-ma.csv"

# Png files generated from CSV file
armAvgAPILatencyPng="arm-api-latency.png"
aspAvgAPILatencyPng="asp-api-latency.png"
avgCpuOrchAppPng="avg-cpu-usage-ma.png"
maxCpuOrchAppPng="max-cpu-usage-ma.png"
avgRamOrchAppPng="avg-ram-usage-ma.png"
maxRamOrchAppPng="max-ram-usage-ma.png"

# Function to display usage help
usage() {
  echo "This script scales the number of concurrent users linearly, \
and then measures the ARM API and ASP latency, and also Maestro-App-System Namespace resource usage during the process"
  echo "Usage: $0 [options] [--] [arguments]"
  echo
  echo "Options:"
  echo "  -u VALUE      Keycloak username, default all-groups-example-user"
  echo "  -p VALUE      Keycloak password, default ChangeMeOn1stLogin!"
  echo "  -f VALUE      Orch FQDN, default integration12.maestro.intel.com"
  echo "  -o VALUE      Observability API credentials base64 encoded"
  echo "  -a VALUE      App ID to be used for running API Latency Checks. This is exactly same ID as the Rancher Fleet Bundle ID of the APP"
  echo "  -r VALUE      App endpoints accessed by each user"
  echo "  -h            Print this help menu"

  echo
  echo "Example:"
  echo "  $0 -a b-8f3c9900-6d02-5e84-9e8d-edf2606d1810 -o Z2djOmdnYzEyMw=="
  exit 1
}

readInputArgs() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
    -u | --user)
      echo "Username: $2"
      enspUser=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -p | --password)
      echo "Password: $2"
      enspPassword=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -f | --fqdn)
      echo "FQDN: $2"
      clusterFqdn=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -o | --observability-api-cred)
      echo "Observability API Credentials base64 encoded: $2"
      observabilityApiCredentials=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -a | --app-id)
      echo "APP ID: $2"
      appID=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -r | --asp-endpoints-per-user)
      echo "Asp Endpoints per User: $2"
      appEndpointsPerUser=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;    
    -h | --help)
      usage
      shift
      ;;
    *)
      echo "Unknown option: $1"
      shift # Shift past the unknown option
      ;;
    esac
  done
}

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
  users=$1
  echo "Running API latency tests... it will take a few mins to capture the results"
  initializeOrchAPIToken
  # Run API Latency Tests
  k6 run --env MY_HOSTNAME="$clusterFqdn" --env API_TOKEN="$orchAPIToken" --env APP_ID="$appID" --env USERS="$users" --env APPS_PER_USER="$appEndpointsPerUser" ./arm-asp-api-latency.js -q --no-thresholds --summary-export="$tmpJSONFile"
  totalChecksPass=$(jq '.root_group.checks."status is OK".passes' "$tmpJSONFile")
  totalChecksFail=$(jq '.root_group.checks."status is OK".fails' "$tmpJSONFile")
  avgArmApiDuration=$(jq '.metrics."http_req_duration{type:armAPI}".avg' "$tmpJSONFile")
  avgArmAspProxyAccessDuration=$(jq '.metrics."http_req_duration{type:containerAppProxy}".avg' "$tmpJSONFile")
  # Capture the result to a csv file to be used for plotting later
  echo "$users,$avgArmApiDuration" >>"$resultsDirectory/$armAvgAPILatencyCsv"
  echo "$users,$avgArmAspProxyAccessDuration" >>"$resultsDirectory/$aspAvgAPILatencyCsv"
  echo "API Latency Check Results: Passed Checks: $totalChecksPass Failed Checks: $totalChecksFail, users: $users, appEndpointsPerUser: $appEndpointsPerUser, Avg ARM API Duration: $avgArmApiDuration, Avg App Proxy Access Duration: $avgArmAspProxyAccessDuration"
}

collectMetric() {
  input=$1
  value="$2"
  query=$3
  resultFile=$4

  value=$(curl -s "$prometheusURL/query" --header "Authorization: Basic ${observabilityApiCredentials}" --data-urlencode "query=$query" | jq -r '.data.result[0].value[1]' 2>/dev/null)
  if [ "$value" != "" ]; then
    echo "$input,$value" >>"$resultsDirectory/$resultFile"
  fi
}

collectObservabilityMetrics() {
  if [ "$observabilityApiCredentials" == "" ]; then
    echo "observabilityApiCredentials is nil, cannot collect observability metrics"
    return
  fi
  users=$1
  secondSuffix="s"
  timeDurationInSec=$2$secondSuffix

  collectMetric "$users" "avgCpu" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_cpu_utilization_ratio{k8s_namespace_name=\"orch-app\"}[$timeDurationInSec]))" $avgCpuOrchAppCsv
  collectMetric "$users" "maxCpu" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_cpu_utilization_ratio{k8s_namespace_name=\"orch-app\"}[$timeDurationInSec]))" $maxCpuOrchAppCsv

  collectMetric "$users" "avgMem" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_memory_usage_bytes{k8s_namespace_name=\"orch-app\"}[$timeDurationInSec]))" $avgRamOrchAppCsv
  collectMetric "$users" "maxMem" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_memory_usage_bytes{k8s_namespace_name=\"orch-app\"}[$timeDurationInSec]))" $maxRamOrchAppCsv
}

plotGraph() {
  # Define the input CSV file and the output image file
  inputCsv="$resultsDirectory/$1"
  outputPng="$resultsDirectory/$2"
  xLabel=$3
  yLabel=$4
  yRangeMin=$5
  yRangeMax=$6
  yAxisScaleFactor=$7
  graphTitle=$8

  # If csv files exists, then generate graphs before exiting the script
  if [ ! -f "$inputCsv" ]; then
    return
  fi
  echo "Plotting graph for $inputCsv"

  # Generate the graph using gnuplot
  gnuplot -persist <<-EOFMarker
    set datafile separator ","
    set terminal png size 800,600
    set output "$outputPng"
    set title "$graphTitle"
    set xlabel "$xLabel"
    set ylabel "$yLabel"
    set ytics scale $yAxisScaleFactor
    set yrange [$yRangeMin:$yRangeMax]
    plot "$inputCsv" using 1:(\$2*$yAxisScaleFactor) with linespoints title "$graphTitle"
EOFMarker

  echo "Graph generated for $graphTitle, output file: $outputPng"
}

cleanup() {
  # Remove temp file used to capture transient results
  rm -f "$tmpJSONFile"

  plotGraph "$armAvgAPILatencyCsv" "$armAvgAPILatencyPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "avg-api-latency-in-ms" 0 2000 1 "arm-api-latency"
  plotGraph "$aspAvgAPILatencyCsv" "$aspAvgAPILatencyPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "avg-api-latency-in-ms" 0 3000 1 "asp-api-latency"
  plotGraph "$avgCpuOrchAppCsv" "$avgCpuOrchAppPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "avg-cpu-usage" 0 5 1 "avg-cpu-usage--ma"
  plotGraph "$maxCpuOrchAppCsv" "$maxCpuOrchAppPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "max-cpu-usage" 0 5 1 "max-cpu-usage--ma"
  plotGraph "$avgRamOrchAppCsv" "$avgRamOrchAppPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "avg-ram-usage-in-MB" 500 10000 0.000001 "avg-ram-usage--ma"
  plotGraph "$maxRamOrchAppCsv" "$maxRamOrchAppPng" "total-concurrent-users-with-$appEndpointsPerUser-app-endpoints-per-user" "max-ram-usage-in-MB" 500 10000 0.000001 "max-ram-usage--ma"
}

# Trap multiple signals
trap cleanup EXIT SIGINT SIGTERM

# Read input arguments
readInputArgs "$@"

if [ -z "$appID"  ] || [ -z "$observabilityApiCredentials" ]; then
  echo "appID: $appID or ObservabilityApiCredentials: $observabilityApiCredentials is nil, can't proceed with the test..."
  exit 1
fi

# Create directory to store results
mkdir -p "$resultsDirectory"

testStartTime=$(date +%s)
runApiLatencyChecks 10
testEndTime=$(date +%s)
collectObservabilityMetrics 10 $((testEndTime - testStartTime))

testStartTime=$(date +%s)
runApiLatencyChecks 15
testEndTime=$(date +%s)
collectObservabilityMetrics 15 $((testEndTime - testStartTime))

testStartTime=$(date +%s)
runApiLatencyChecks 20
testEndTime=$(date +%s)
collectObservabilityMetrics 20 $((testEndTime - testStartTime))

testStartTime=$(date +%s)
runApiLatencyChecks 25
testEndTime=$(date +%s)
collectObservabilityMetrics 25 $((testEndTime - testStartTime))

testStartTime=$(date +%s)
runApiLatencyChecks 30
testEndTime=$(date +%s)
collectObservabilityMetrics 30 $((testEndTime - testStartTime))
