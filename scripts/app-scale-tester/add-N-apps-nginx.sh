#!/bin/bash
set -eu

if [ $# == 0 ]; then
  echo "specify number of apps to setup"
  exit 1
fi

# Initialize some global variables and constants
clusterFqdn=integration14.espd.infra-host.com # Change if needed
# clusterFqdn=kind.internal # Change if needed
enspUser=intel-itep-user && enspPassword=ChangeMeOn1stLogin!
prometheusURL="https://observability-admin.${clusterFqdn}/api/datasources/uid/orchestrator-mimir/resources/api/v1"
prometheusTestURL="https://observability-admin.${clusterFqdn}/api/datasources/uid/orchestrator-mimir/health"
project=itep
targetClusterLabel="default-extension=baseline"

## App install specific variables
totalApps=$1           # Total apps to install - input argument
batchSize=10           # How many apps to install in parallel?
interval=10
pageSize=100           # Adjust pageSize as needed for querying elements on APIs
offset=0
runTests=1

## Token specific variables
orchAPIToken=""          # Orchestrator API Token
observabilityApiCredentials="" # Provide a valid token for the test.

## App install specific variables
containerDeploymentPackageName="nginx-app"
containerAppName="nginx"
containerAppVersion="0.1.0"
containerAppProfileName="default-profile"
containerAppPublisherName="default"
totalContainerAppInstancesPerCluster=3

# Container deployment template -- currently not used but kept for reference.
# The template approach gets complex when we have a Deployment with multiple apps.
# shellcheck disable=SC2089
containerAppDeploymentTemplate='{"appName":"%s","appVersion":"%s","name":"dummy-app-%d","profileName":"%s","publisherName":"%s","targetClusters":[{"appName":"%s", "labels":{"target":"scale"}}],"displayName":"dummy-app-%d","deploymentType":"auto-scaling","overrideValues":[]}'

# CSV Files to capture result
currentTimeStamp=$(date -u +"%Y%m%d-%H%M%S")
resultsDirectory="./test-results/$currentTimeStamp"
tmpJSONFile="$resultsDirectory/tmp.json"
appInstallTimeCsv="app-install-time.csv"
admApiLatencyCsv="adm-api-latency.csv"
#armApiLatencyCsv="arm-api-latency.csv"
avgCpuMaestroAppSystemCsv="avg-cpu-usage-ma.csv"
avgCpuCattleSystemCsv="avg-cpu-usage-cattle-system.csv"
avgCpuCattleFleetSystemCsv="avg-cpu-usage-cattle-fleet-system.csv"
maxCpuMaestroAppSystemCsv="max-cpu-usage-ma.csv"
maxCpuCattleSystemCsv="max-cpu-usage-cattle-system.csv"
maxCpuCattleFleetSystemCsv="max-cpu-usage-cattle-fleet-system.csv"
avgRamMaestroAppSystemCsv="avg-ram-usage-ma.csv"
avgRamCattleSystemCsv="avg-ram-usage-cattle-system.csv"
avgRamCattleFleetSystemCsv="avg-ram-usage-cattle-fleet-system.csv"
maxRamMaestroAppSystemCsv="max-ram-usage-ma.csv"
maxRamCattleSystemCsv="max-ram-usage-cattle-system.csv"
maxRamCattleFleetSystemCsv="max-ram-usage-cattle-fleet-system.csv"

# Png files generated from CSV file
appInstallTimePng="app-install-time.png"
admApiLatencyPng="adm-api-latency.png"
#armApiLatencyPng="arm-api-latency.png"
avgCpuMaestroAppSystemPng="avg-cpu-usage-ma.png"
avgCpuCattleSystemPng="avg-cpu-usage-cattle-system.png"
avgCpuCattleFleetSystemPng="avg-cpu-usage-cattle-fleet-system.png"
maxCpuMaestroAppSystemPng="max-cpu-usage-ma.png"
maxCpuCattleSystemPng="max-cpu-usage-cattle-system.png"
maxCpuCattleFleetSystemPng="max-cpu-usage-cattle-fleet-system.png"
avgRamMaestroAppSystemPng="avg-ram-usage-ma.png"
avgRamCattleSystemPng="avg-ram-usage-cattle-system.png"
avgRamCattleFleetSystemPng="avg-ram-usage-cattle-fleet-system.png"
maxRamMaestroAppSystemPng="max-ram-usage-ma.png"
maxRamCattleSystemPng="max-ram-usage-cattle-system.png"
maxRamCattleFleetSystemPng="max-ram-usage-cattle-fleet-system.png"

appOrchEndpoint="https://api.${clusterFqdn}"

# Function to display usage help
usage() {
  echo "Usage: $0 [options] [--] [arguments]"
  echo
  echo "Options:"
  echo "  -u VALUE      Keycloak username, default all-groups-example-user"
  echo "  -p VALUE      Keycloak password, default ChangeMeOn1stLogin!"
  echo "  -f VALUE      Orch FQDN, default integration12.espd.infra-host.com"
  echo "  -b VALUE      Cluster install batch size, default 10"
  echo "  -o VALUE      Observability API credentials base64 encoded"
  echo "  -a VALUE      Apps per cluster, default 3"
  echo "  -d VALUE      Deployment package name to deploy, default dummy-app-package"
  echo "  -v VALUE      Deployment package version to deploy, default 0.0.1"
  echo "  -g VALUE      Don't run tests, only regenerate graphs using the specified results directory"
  echo "  -m VALUE      project, default itep"
  echo "  -h            Print this help menu"

  echo
  echo "Example:"
  echo "  $0 -a 10 -b 2"
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
    -h | --help)
      usage
      shift
      ;;
    -f | --fqdn)
      echo "FQDN: $2"
      clusterFqdn=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -b | --batch-size)
      echo "Cluster Install batch size: $2"
      batchSize=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -o | --observability-api-cred)
      echo "Observability API Credentials base64 encoded: $2"
      observabilityApiCredentials=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -a | --total-apps-per-cluster)
      echo "Apps per enic: $2"
      totalApps=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -d | --deployment-package-name)
      echo "Deployment package name: $2"
      containerDeploymentPackageName=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -v | --deployment-package-version)
      echo "Deployment-package-version: $2"
      containerAppVersion=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -m | --project)
      echo "Project: $2"
      project=$2
      shift # Shift past the option
      shift # Shift past the value
      ;;
    -g)
      echo "Results directory: $2"
      resultsDirectory=$2
      exit 0
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
    orchAPIToken=$(curl -s --location --request POST https://keycloak.${clusterFqdn}/realms/master/protocol/openid-connect/token \
      -H 'Content-Type: application/x-www-form-urlencoded' \
      -d "username=${enspUser}" \
      -d "password=${enspPassword}" \
      -d "grant_type=password" \
      -d "client_id=system-client" \
      -d "scope=openid" |
      jq -r '.access_token' 2>/dev/null)
}

catalogLogin() {
  CATALOG_ARGS="--deployment-endpoint ${appOrchEndpoint} --catalog-endpoint ${appOrchEndpoint}"
  catalog ${CATALOG_ARGS} logout

  catalog ${CATALOG_ARGS} login --client-id=system-client --trust-cert=true --keycloak https://keycloak.${clusterFqdn}/realms/master ${enspUser} ${enspPassword}
}

# Provide customized deployment for different packages (particularly target labels)
# Arguments are deploymentPackageName, deploymentPackageVersion
catalogDeploy() {
  NAME=$1
  VERSION=$2

  CATALOG_ARGS="--deployment-endpoint ${appOrchEndpoint} --catalog-endpoint ${appOrchEndpoint} --project ${project}"
  case ${NAME} in
    dummy-app-package)
      catalog ${CATALOG_ARGS} create deployment \
        ${NAME} ${VERSION} --application-label dummy-app.$targetClusterLabel
    ;;
    ten-dummy-apps)
      catalog ${CATALOG_ARGS} create deployment \
        ${NAME} ${VERSION} --application-label dummy-app-1.$targetClusterLabel \
        --application-label dummy-app-2.$targetClusterLabel \
        --application-label dummy-app-3.$targetClusterLabel \
        --application-label dummy-app-4.$targetClusterLabel \
        --application-label dummy-app-5.$targetClusterLabel \
        --application-label dummy-app-6.$targetClusterLabel \
        --application-label dummy-app-7.$targetClusterLabel \
        --application-label dummy-app-8.$targetClusterLabel \
        --application-label dummy-app-9.$targetClusterLabel \
        --application-label dummy-app-10.$targetClusterLabel
    ;;
    nginx-app)
      catalog ${CATALOG_ARGS} create deployment \
        ${NAME} ${VERSION} --application-label nginx.$targetClusterLabel
    ;;
    *)
      echo "ERROR: No deploy template for deployment package ${NAME}"
      exit 1
    ;;
  esac
}

catalogUpload() {
  catalogLogin

  CATALOG_ARGS="--deployment-endpoint ${appOrchEndpoint} --catalog-endpoint ${appOrchEndpoint} --project ${project}"
  pushd ../../deployment-packages/
  for DPDIR in $(ls -d $containerDeploymentPackageName)
  do
      cd ${DPDIR}; catalog ${CATALOG_ARGS} upload .; cd ..
  done
  popd
}

createContainerDeployment() {
  echo Creating container deployment
  batch=$1
  appDeploymentsCounter=$(( batch * batchSize))

  containerAppInstanceCnt=$appDeploymentsCounter
  while [ "$containerAppInstanceCnt" -lt "$(( (batch + 1) * batchSize ))" ]; do
    # shellcheck disable=SC2090
    # containerAppInstallSpec=$(printf "$containerAppDeploymentTemplate" $containerDeploymentPackageName $containerAppVersion $containerAppInstanceCnt $containerAppProfileName $containerAppPublisherName $containerAppName $containerAppInstanceCnt)
    # echo "App Deployment Spec is $containerAppInstallSpec"
    # curl -s -X POST "https://app-orch.$clusterFqdn/deployment.orchestrator.apis/v1/deployments" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" -d "$containerAppInstallSpec"
    catalogLogin
    catalogDeploy $containerDeploymentPackageName $containerAppVersion
    echo ""
    # shellcheck disable=SC2003
    containerAppInstanceCnt=$(expr $containerAppInstanceCnt + 1)
  done
}

deleteDeployments() {
    arr=()

    totalAppCnt=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq '.totalElements')
    for ((i = 0 ; i < totalAppCnt ; i++ ));
    do
      currDeployment=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments?pageSize=${pageSize}&offset=${offset}" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq -r '.deployments['$i']')
      appName=$(echo $currDeployment | jq -r '.appName')
      deployId=$(echo $currDeployment | jq -r '.deployId')

      if [ "$appName" == "$containerDeploymentPackageName" ]; then
        arr+=("$deployId")
      fi
    done

    for d in "${arr[@]}"
    do
      deployId=$(curl -X DELETE "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments/$d" -H "Authorization: Bearer $orchAPIToken")
      echo "Deleted deployment $d..."
    done
    sleep 2
}

waitForAllAppsToBeRunning() {
  initializeOrchAPIToken
  echo Waiting for all apps to be running

#echo "pagesize print: $pageSize"
  while true; do
    totalAppCnt=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq '.totalElements')
    set +e
    totalCnt=0
    for ((i = 0 ; i < totalAppCnt ; i++ ));
    do
      currDeployment=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments?pageSize=${pageSize}&offset=${offset}" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq -r '.deployments['$i']')
      appName=$(echo $currDeployment | jq -r '.appName')
#echo "appname print : $appName"

      if [ "$appName" == "$containerDeploymentPackageName" ]; then
        totalCnt=$((totalCnt + 1))
#echo "totalCnt print : $totalCnt"
      fi
    done
#echo "totalCnt print out: $totalCnt"
#runningAppCnt=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments?pageSize=${pageSize}&offset=${offset}" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq -r '.deployments[].status.state' | grep -c RUNNING)
    runningCnt=0
    for ((i = 0 ; i < totalAppCnt ; i++ ));
    do
      currDeployment=$(curl -s "$appOrchEndpoint/v1/projects/$project/appdeployment/deployments?pageSize=${pageSize}&offset=${offset}" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" | jq -r '.deployments['$i']')
      appName=$(echo $currDeployment | jq -r '.appName')
      statusVal=$(echo $currDeployment | jq -r '.status.state')
#echo "appname print : $appName"
#echo "statusval print : $statusVal"

      if [[ "$appName" == "$containerDeploymentPackageName" && "$statusVal" == "RUNNING" ]]; then
        runningCnt=$((runningCnt + 1))
#echo "runningCnt print : $runningCnt"
      fi
    done
#echo "runningCnt print out: $runningCnt"
    set -e
    # curl -s "https://app-orch.$clusterFqdn/deployment.orchestrator.apis/v1/summary/deployments_status" -H 'Content-Type: application/json' -H 'accept: application/json' -H "Authorization: Bearer $orchAPIToken" >"$tmpJSONFile" 2>/dev/null
    # Method below does not seem to provide an accurate total app count (see NEX-2988)
    # totalAppCnt=$(jq '.total' "$tmpJSONFile")
    # runningCnt=$(jq '.running' "$tmpJSONFile")
    if [ "$totalCnt" == "$runningCnt" ]; then
      echo "All $totalAppCnt apps are running!!"
      break
    fi

    echo "$runningCnt / $totalCnt apps running. Waiting $interval seconds."
    sleep $interval
  done
}

runApiLatencyChecks() {
  echo "Running API latency tests... it will take a few mins to capture the results"
  initializeOrchAPIToken
  # Run API Latency Tests
  k6 run --env MY_HOSTNAME=$clusterFqdn --env API_TOKEN="$orchAPIToken" --env PROJECT="$project" ../k6-scripts/adm-api-latency.js -q --no-thresholds --summary-export="$tmpJSONFile"
  totalChecksPass=$(jq '.root_group.checks."status is OK".passes' "$tmpJSONFile")
  totalChecksFail=$(jq '.root_group.checks."status is OK".fails' "$tmpJSONFile")
  avgApiDuration=$(jq '.metrics.http_req_duration.avg' "$tmpJSONFile")
  # Capture the result to a csv file to be used for plotting later
  echo "$totalAppCnt,$avgApiDuration" >>"$resultsDirectory/$admApiLatencyCsv"
  echo "API Latency Check Results: Passed Checks: $totalChecksPass Failed Checks: $totalChecksFail, Avg API Duration: $avgApiDuration"
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

collectMetric() {
  input=$1
  value="$2"
  query=$3
  resultFile=$4

  value=$(curl -s "$prometheusURL/query" -H "Authorization: Basic ${observabilityApiCredentials}" --data-urlencode "query=$query" | jq -r '.data.result[0].value[1]' 2>/dev/null)
  if [ "$value" != "" ]; then
    echo "$input,$value" >>"$resultsDirectory/$resultFile"
  fi
}

collectObservabilityMetrics() {
  if [ "$observabilityApiCredentials" == "" ]; then
    echo "observabilityApiCredentials is nil, cannot collect observability metrics"
    return
  fi

  # Check that Prometheus is accessible with the provided credentials
  status=$(curl -s "$prometheusTestURL" -H "Authorization: Basic ${observabilityApiCredentials}" | jq -r '.status')
  if [ "$status" != "OK" ]; then
    echo "Unable to query Prometheus with credentials provided, cannot collect observability metrics"
    return
  fi

  totalApps=$1
  secondSuffix="s"
  timeDurationInSec=$2$secondSuffix

  # Remove "_ratio" suffix due observability name changes (e.g., https://github.com/intel-innersource/frameworks.edge.one-intel-edge.observability.platform-dashboard/pull/15/files)
  collectMetric "$totalApps" "avgCpu" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"maestro-app-system\"}[$timeDurationInSec]))" $avgCpuMaestroAppSystemCsv
  collectMetric "$totalApps" "maxCpu" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"maestro-app-system\"}[$timeDurationInSec]))" $maxCpuMaestroAppSystemCsv
  collectMetric "$totalApps" "avgCpu" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"cattle-system\"}[$timeDurationInSec]))" $avgCpuCattleSystemCsv
  collectMetric "$totalApps" "maxCpu" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"cattle-system\"}[$timeDurationInSec]))" $maxCpuCattleSystemCsv
  collectMetric "$totalApps" "avgCpu" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"cattle-fleet-system\"}[$timeDurationInSec]))" $avgCpuCattleFleetSystemCsv
  collectMetric "$totalApps" "maxCpu" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_cpu_utilization{k8s_namespace_name=\"cattle-fleet-system\"}[$timeDurationInSec]))" $maxCpuCattleFleetSystemCsv

  # Remove "_bytes" suffix due observability name changes (e.g., https://github.com/intel-innersource/frameworks.edge.one-intel-edge.observability.platform-dashboard/pull/15/files)
  collectMetric "$totalApps" "avgMem" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"maestro-app-system\"}[$timeDurationInSec]))" $avgRamMaestroAppSystemCsv
  collectMetric "$totalApps" "maxMem" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"maestro-app-system\"}[$timeDurationInSec]))" $maxRamMaestroAppSystemCsv
  collectMetric "$totalApps" "avgMem" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"cattle-system\"}[$timeDurationInSec]))" $avgRamCattleSystemCsv
  collectMetric "$totalApps" "maxMem" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"cattle-system\"}[$timeDurationInSec]))" $maxRamCattleSystemCsv
  collectMetric "$totalApps" "avgMem" "sum by(k8s_namespace_name)(avg_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"cattle-fleet-system\"}[$timeDurationInSec]))" $avgRamCattleFleetSystemCsv
  collectMetric "$totalApps" "maxMem" "sum by(k8s_namespace_name)(max_over_time(k8s_pod_memory_usage{k8s_namespace_name=\"cattle-fleet-system\"}[$timeDurationInSec]))" $maxRamCattleFleetSystemCsv
}

cleanup() {
  # Remove temp file used to capture transient results
  rm -f "$tmpJSONFile"

  plotGraph "$appInstallTimeCsv" "$appInstallTimePng" "Deployment #" "Seconds until Running on all edges" 0 500 1 "Time to Running on 1K edges, per Deployment"
  plotGraph "$admApiLatencyCsv" "$admApiLatencyPng" "Total Deployments" "Avg API latency in ms" 0 1000 1 "Avg ADM API Latency by # of Deployments (1K edges)"
  plotGraph "$avgCpuMaestroAppSystemCsv" "$avgCpuMaestroAppSystemPng" "Total Deployments" "Avg CPU usage" 0 10 1 "Avg App Orch CPU usage by # of Deployments (1K edges)"
  plotGraph "$avgCpuCattleSystemCsv" "$avgCpuCattleSystemPng" "Total Deployments" "Avg CPU usage" 0 10 1 "Avg Rancher CPU usage by # of Deployments (1K edges)"
  plotGraph "$avgCpuCattleFleetSystemCsv" "$avgCpuCattleFleetSystemPng" "Total Deployments" "Avg CPU usage" 0 40 1 "Avg Fleet CPU usage by # of Deployments (1K edges)"
  plotGraph "$maxCpuMaestroAppSystemCsv" "$maxCpuMaestroAppSystemPng" "Total Deployments" "Max CPU usage" 0 10 1 "Max App Orch CPU usages by # of Deployments (1K edges)"
  plotGraph "$maxCpuCattleSystemCsv" "$maxCpuCattleSystemPng" "Total Deployments" "Max CPU usage" 0 10 1 "Max Rancher CPU usage by # of Deployments (1K edges)"
  plotGraph "$maxCpuCattleFleetSystemCsv" "$maxCpuCattleFleetSystemPng" "Total Deployments" "Max CPU usage" 0 40 1 "Max Fleet CPU usage by # of Deployments (1K edges)"
  plotGraph "$avgRamMaestroAppSystemCsv" "$avgRamMaestroAppSystemPng" "Total Deployments" "Avg RAM usage in MB" 500 10000 0.000001 "Avg App Orch RAM usage by # of Deployments (1K edges)"
  plotGraph "$avgRamCattleSystemCsv" "$avgRamCattleSystemPng" "Total Deployments" "Avg RAM usage in MB" 500 40000 0.000001 "Avg Rancher RAM usage by # of Deployments (1K edges)"
  plotGraph "$avgRamCattleFleetSystemCsv" "$avgRamCattleFleetSystemPng" "Total Deployments" "Avg RAM usage in MB" 500 10000 0.000001 "Avg Fleet RAM usage by # of Deployments (1K edges)"
  plotGraph "$maxRamMaestroAppSystemCsv" "$maxRamMaestroAppSystemPng" "Total Deployments" "Max RAM usage in MB" 500 10000 0.000001 "Max App Orch RAM usage by # of Deployments (1K edges)"
  plotGraph "$maxRamCattleSystemCsv" "$maxRamCattleSystemPng" "Total Deployments" "Max RAM usage in MB" 500 40000 0.000001 "Max Rancher RAM usage by # of Deployments (1K edges)"
  plotGraph "$maxRamCattleFleetSystemCsv" "$maxRamCattleFleetSystemPng" "Total Deployments" "Max RAM usage in MB" 500 10000 0.000001 "Max Fleet RAM usage by # of Deployments (1K edges)"
}

########################### Main Script Starts here ###########################

# Trap multiple signals
trap cleanup EXIT SIGINT SIGTERM

# Read input arguments
readInputArgs "$@"

if [ "$totalApps" == 0 ]; then
  echo "No apps to be setup, exit.."
  exit 0
fi

# Initialize Orchestrator Keycloak Token to be used for API access
initializeOrchAPIToken
echo "Initialized Orchestrator Keycloak Token"

deleteDeployments
echo "Deleted previous deployments"

# Create directory to store results
mkdir -p "$resultsDirectory"
echo "Created directory $resultsDirectory"

if [ "$totalApps" -lt "$batchSize" ]; then
  batchSize=$totalApps
fi

echo "Creating $totalApps apps..."

catalogUpload
echo "Uploaded deployment packages"

totalBatches=$((totalApps / batchSize))
remainder=$((totalApps % batchSize))
if [ $remainder -ne 0 ]; then
  # shellcheck disable=SC2003
  totalBatches=$(expr $totalBatches + 1)
fi

counter=0

# Install apps
while [ $counter -lt "$totalBatches" ]; do
  testStartTime=$(date +%s)

  # Initialize Orch API Token again to be safe. Token could have possibly expired while
  # we installed previous batch of clusters or could expire shortly
  initializeOrchAPIToken

  createContainerDeployment $counter

  # Wait for all apps to running
  appInstallStartTime=$(date +%s)
  waitForAllAppsToBeRunning

  appInstallEndTime=$(date +%s)
  echo Total time for all apps to be running at iteration:$counter: $((appInstallEndTime - appInstallStartTime)) seconds

  # Capture the result to a csv file to be used for plotting later
  echo "$totalAppCnt,$((appInstallEndTime - appInstallStartTime))" >>"$resultsDirectory/$appInstallTimeCsv"

  # Run API Latency Checks
  runApiLatencyChecks
  testEndTime=$(date +%s)
  echo "Total Test Run Time for Batch$counter: $((testEndTime - testStartTime))s"

  # Collect metrics from observability APIs
  collectObservabilityMetrics "$totalAppCnt" $((testEndTime - testStartTime))

  # shellcheck disable=SC2003
  counter=$(expr $counter + 1)
done

