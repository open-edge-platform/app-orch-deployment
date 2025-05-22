import {check, fail} from 'k6';
import http from 'k6/http';
import exec from "k6/execution";
import ws from 'k6/ws';

let timeDurationBetweenScenarioInMin = 10;
let containerAppEndpointAccessLatencyTestStartTime = 0;
let users = parseInt(__ENV.USERS) || 10
let appsPerUser = parseInt(__ENV.APPS_PER_USER) || 100
// Dynamically create the scenario configuration for K6 runner
let scenariosDynamic = {};
let numOfScenarios = 0;
let sessionDuration = 1000;

// Scenario for testing API Latency of Container Application access via App Proxys
scenariosDynamic[numOfScenarios++] = {
    executor: 'per-vu-iterations',
    exec: 'vmVNCAccessLatencyTest',
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
        'ws_connecting{type:vncProxy}': ['p(95)<3000', 'avg<2500'], // 95% of requests from AppProxy should be take below 3000ms, avg is less 2500ms
        'ws_session_duration{type:vncProxy}': ['p(95)<3000', 'avg<2500'], // http errors should be less than 1%
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

// Define Http header for VNC proxy tests
const httpOptionsContainerAppProxy = {
    headers: {
        cookie: `keycloak-token=${__ENV.API_TOKEN}`,
        'Content-Type': 'application/json',
    },
    tags: {type: 'vncProxy'},
};
export function vmVNCAccessLatencyTest() {
    // cluster - app - VM ID list
    var numOfTargetVMs = {}

    // get Cluster List
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
            if (_cluster.id !== "cluster1") {
                clusterNameList.push(_cluster.id);
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

    // get URL from ARM
    let vncEndpoint = Array();
    let startIdx = parseInt((clusterNameList.length / users) * (exec.vu.idInInstance-1))
    for (let i = startIdx; i < (startIdx + appsPerUser) && i < clusterNameList.length; i++) {
        let _name = clusterNameList[i]
        let response = http.get(`https://app-orch.${__ENV.MY_HOSTNAME}/resource.orchestrator.apis/v2/workloads/${__ENV.APP_ID}/${_name}`, httpOptionsARM);
        if (
            !check(response, {
                "status is OK": (r) => r && r.status === 200,
            })
        ) {
            fail(`failed to get app endpoints, status not 200 for cluster: ${_name}`);
            return;
        }

        let _appWorkloads = response.json().appWorkloads
        _appWorkloads.forEach(function (obj) {
            if (obj.type === "TYPE_VIRTUAL_MACHINE" ) {
                let _vmID = obj.id
                let vncAPIResponse = http.get(`https://app-orch.${__ENV.MY_HOSTNAME}/resource.orchestrator.apis/v2/workloads/virtual-machines/${__ENV.APP_ID}/${_name}/${_vmID}/vnc`, httpOptionsARM);
                if (
                    !check(vncAPIResponse, {
                        "status is OK": (r) => r && r.status === 200,
                    })
                ) {
                    fail(`failed to get app vnc address, status not 200 for cluster: ${_name}`);
                    return;
                }

                vncEndpoint.push(vncAPIResponse.json().address)
            }
        })
    }

    vncEndpoint.forEach(function (obj) {
        const params = { headers: {
            "Origin": `https://web-ui.${__ENV.MY_HOSTNAME}`,
            cookie: `keycloak-token=${__ENV.API_TOKEN}`,
            },tags: {type: 'vncProxy'}};

        const res = ws.connect(obj, params, function (socket) {

            socket.on('open', function open() {
                console.log(`VU ${__VU}: connected / ${obj.toString()}`);
            });

            socket.on('binaryMessage', (message) => {
                var rMsg = new Uint8Array(message)
                var msg = String.fromCharCode.apply(null, rMsg)
                if (msg.toString() !== "RFB 003.008\n") {
                    fail(`unexpected message arrived - should be RFB 003.008 but received ${msg.toString()}`)
                }
                socket.close()
            });

            socket.setTimeout(function () {
              fail("session over sessionDuration", sessionDuration)
            }, sessionDuration);

            socket.setTimeout(function () {
              socket.close();
              fail(`Closing the socket forcefully 3s after graceful LEAVE`)
            }, sessionDuration + 30000);
        });

        check(res, { 'Connected successfully': (r) => r && r.status === 101 });
    });
}