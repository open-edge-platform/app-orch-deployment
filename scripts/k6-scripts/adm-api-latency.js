import { check } from 'k6';
import http from 'k6/http';

// Define some constants and global variables.
let timeDurationBetweenScenarioInMin = 1;
let deploymentListFromADMStartTime = 0;
let deploymentSummaryFromAdmStartTime = timeDurationBetweenScenarioInMin + deploymentListFromADMStartTime
let clusterListFromAdmStartTime = timeDurationBetweenScenarioInMin + deploymentSummaryFromAdmStartTime

// Dynamically create the scenario configuration for K6 runner
let scenariosDynamic = {};
let numOfScenarios = 0;

// Scenario for querying deployment list from ADM
scenariosDynamic[numOfScenarios++] = {
    executor: 'constant-vus',
    exec: 'deploymentListFromADM',
    vus: 10,
    startTime: deploymentSummaryFromAdmStartTime.toString() + 'm',
    duration: timeDurationBetweenScenarioInMin.toString() + 'm',
}

// Scenario for querying deployment summary from ADM
scenariosDynamic[numOfScenarios++] = {
    executor: 'constant-vus',
    exec: 'deploymentSummaryFromADM',
    vus: 10,
    startTime: deploymentListFromADMStartTime.toString() + 'm',
    duration: timeDurationBetweenScenarioInMin.toString() + 'm',
}

// Scenario for querying cluster list from ADM
scenariosDynamic[numOfScenarios++] = {
    executor: 'constant-vus',
    exec: 'clusterListFromADM',
    vus: 10,
    startTime: clusterListFromAdmStartTime.toString() + 'm',
    duration: timeDurationBetweenScenarioInMin.toString() + 'm',
}



// Set k6 test options
export const options = {
    scenarios: scenariosDynamic,
    // thresholds: Defines the checks to perform at the end of the tests
    thresholds: {
        http_req_failed: ['rate<0.01'], // http errors should be less than 1%
        'http_req_duration{type:admApiStats}': ['p(95)<1000', 'avg<500'], // 95% of Cluster Query requests from ECM should be take below 1000ms, avg is less 500ms
        'http_req_failed{type:admApiStats}': ['rate<0.01'], // http errors should be less than 1%
    },
};

// Define Http header for App List query from ADM
const httpOptionsAdmQuery = {
    headers: {
        Authorization: `Bearer ${__ENV.API_TOKEN}`,
        'Content-Type': 'application/json',
    },
    tags: {type: 'admApiStats'},
};

// deploymentListFromADM queries the deployment List from ADM NB API
export function deploymentListFromADM() {
    let response = http.get(`https://api.${__ENV.MY_HOSTNAME}/v1/projects/${__ENV.PROJECT}/appdeployment/deployments`, httpOptionsAdmQuery);
    check(response, {
        "status is OK": (r) => r && r.status === 200,
    });
}

// deploymentSummaryFromADM queries the deployment Summary of deployments ADM NB API
export function deploymentSummaryFromADM() {
    let response = http.get(`https://api.${__ENV.MY_HOSTNAME}/v1/projects/${__ENV.PROJECT}/summary/deployments_status`, httpOptionsAdmQuery);
    check(response, {
        "status is OK": (r) => r && r.status === 200,
    });
}

// clusterListFromADM queries the clusters from ADM NB API
export function clusterListFromADM() {
    let response = http.get(`https://api.${__ENV.MY_HOSTNAME}/v1/projects/${__ENV.PROJECT}/appdeployment/clusters`, httpOptionsAdmQuery);
    check(response, {
        "status is OK": (r) => r && r.status === 200,
    });
}
