import {check, fail} from 'k6';
import http from 'k6/http';
import exec from "k6/execution";

/*
HOW TO RUN
----------

export CO_CLUSTER_FQDN=integration12.maestro.intel.com # Change per your requirement

export API_TOKEN=$(curl -k -s -X POST "https://keycloak.${CO_CLUSTER_FQDN}/realms/master/protocol/openid-connect/token" -d "username=all-groups-example-user" -d 'password=ChangeMeOn1stLogin!' -d "grant_type=password" -d "client_id=system-client" -d "scope=openid" | jq -r ".access_token")


k6 run --env MY_HOSTNAME=integration12.maestro.intel.com --env API_TOKEN=$API_TOKEN --env APP_ID="b-8f3c9900-6d02-5e84-9e8d-edf2606d1810" --env USERS=10 --env APPS_PER_USER=100 asp-container-api-latency.js

NOTE on Options to the script
- API_TOKEN: Keycloak JWT token
- APP_ID: Fleet Bundle ID Of the App
- USERS: Number of concurrent users
- APPS_PER_USER: No of app endpoints each user will access during the test
 */



// Define some constants and global variables.
let timeDurationBetweenScenarioInMin = 10;
let containerAppEndpointAccessLatencyTestStartTime = 0;
let users = parseInt(__ENV.USERS) || 10
let appsPerUser = parseInt(__ENV.APPS_PER_USER) || 100
// Dynamically create the scenario configuration for K6 runner
let scenariosDynamic = {};
let numOfScenarios = 0;

// Scenario for testing API Latency of Container Application access via App Proxys
scenariosDynamic[numOfScenarios++] = {
    executor: 'per-vu-iterations',
    exec: 'containerAppProxyLatencyTest',
    vus: users,
    iterations: 1,
    startTime: containerAppEndpointAccessLatencyTestStartTime.toString() + 'm',
    maxDuration: timeDurationBetweenScenarioInMin.toString() + 'm',
}


// Set k6 test options
export const options = {
    scenarios: scenariosDynamic,
    // thresholds: Defines the checks to perform at the end of the tests
    thresholds: {
        http_req_failed: ['rate<0.01'], // http errors should be less than 1%
        'http_req_duration{type:armAPI}': ['p(95)<1000', 'avg<750'], // 95% of  API from ARM should be take below 1000ms, avg is less 500ms
        'http_req_failed{type:armAPI}': ['rate<0.01'], // http errors should be less than 1%
        'http_req_duration{type:containerAppProxy}': ['p(95)<3000', 'avg<2500'], // 95% of requests from AppProxy should be take below 3000ms, avg is less 2500ms
        'http_req_failed{type:containerAppProxy}': ['rate<0.01'], // http errors should be less than 1%
    },
};

// Define Http header for ARM API Access
const httpOptionsARM = {
    headers: {
        Authorization: `Bearer ${__ENV.API_TOKEN}`,
        'Content-Type': 'application/json',
    },
    tags: {type: 'armAPI'},
};

// Define Http header for ContainerAppProxy tests
const httpOptionsContainerAppProxy = {
    headers: {
        cookie: `keycloak-token=${__ENV.API_TOKEN}`,
        'Content-Type': 'application/json',
    },
    tags: {type: 'containerAppProxy'},
};

// containerAppProxyLatencyTest tests API latency of Container Applications via the App Proxy
export function containerAppProxyLatencyTest() {
    // First get the list of clusters names
    let clusterNameList = Array();
    let offSet = 0;
    let defaultPageSize = 100
    let response = http.get(`https://app-orch.${__ENV.MY_HOSTNAME}/deployment.orchestrator.apis/v1/clusters?pageSize=${defaultPageSize}&offset=${offSet}`, httpOptionsARM);
    if (
        !check(response, {
            "status is OK": (r) => r && r.status === 200,
        })
    ) {
        fail('failed to get clusters, status not 200');
        return;
    }
    let totalClusterPages = parseInt(response.json().totalElements / defaultPageSize)
    if ((parseInt(response.json().totalElements) % defaultPageSize) !== 0) {
        totalClusterPages++;
    }
    while (totalClusterPages >= 0) {
        // TODO: Use totalElements to calculate the pageSize
        let clusters = response.json().clusters;

        // Populate all cluster names
        for (let _cluster of clusters) {
            if (_cluster.name !== "cluster1") {
                clusterNameList.push(_cluster.name);
            }
        }
        totalClusterPages--;
        if (totalClusterPages === 0) {
            break;
        } else {
            offSet = offSet + defaultPageSize;
            response = http.get(`https://app-orch.${__ENV.MY_HOSTNAME}/deployment.orchestrator.apis/v1/clusters?pageSize=${defaultPageSize}&offset=${offSet}`, httpOptionsARM);
        }
    }
    console.log(`Cluster length: `, clusterNameList.length)

    // Now for the given AppId get the endpoints from each cluster
    // Distribute the clusters across all concurrent users
    let appEndpoints = Array();

    let startIdx = parseInt((clusterNameList.length / users) * (exec.vu.idInInstance-1))

    for (let i = startIdx; i < (startIdx + appsPerUser) && i < clusterNameList.length; i++) {
        let _name = clusterNameList[i]
        let response = http.get(`https://app-orch.${__ENV.MY_HOSTNAME}/resource.orchestrator.apis/v2/endpoints/${__ENV.APP_ID}/${_name}`, httpOptionsARM);
        if (
            !check(response, {
                "status is OK": (r) => r && r.status === 200,
            })
        ) {
            fail(`failed to get app endpoints, status not 200 for cluster: ${_name}`);
            return;
        }
        // Populate the App Endpoints here
        let _appEndpoints = response.json().appEndpoints
        _appEndpoints.forEach(function (obj) {
            if (obj.endpointStatus.state !== "STATE_READY") {
                let appId = obj.id
                fail(`App Endpoint not in ready state for cluster: ${_name}, appId: ${appId}`)
            }
            appEndpoints.push(obj.ports[0].serviceProxyUrl)
        })
    }
    console.log(`Total App Endpoints to verify: ${appEndpoints.length}`)

    // Finally access the App Endpoints via the App Proxy
    for (let _endpoint of appEndpoints) {
        let response = http.get(`${_endpoint}`, httpOptionsContainerAppProxy);
        if (
            !check(response, {
                "status is OK": (r) => r && r.status === 200,
        })) {
            console.log(`failed ${_endpoint}`)
        }
    }
}
