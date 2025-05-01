package deployment

import (
	"fmt"
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/test/utils"
)

var methodStatusMap = map[string]map[string]int{
	"listDeployments": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusBadRequest,
	},
	"listDeploymentsPerCluster": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"getDeleteDeployment": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusOK,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusBadRequest,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"getDeploymentsStatus": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
	"listDeploymentClusters": {
		http.MethodGet:    http.StatusOK,
		http.MethodDelete: http.StatusMethodNotAllowed,
		http.MethodPatch:  http.StatusMethodNotAllowed,
		http.MethodPut:    http.StatusMethodNotAllowed,
		http.MethodPost:   http.StatusMethodNotAllowed,
	},
}

func (s *TestSuite) testMethod(url string, methodMap map[string]int, updateDeployID bool) {
	for method, expectedStatus := range methodMap {
		res, err := utils.CallMethod(url, method, token, projectID)
		s.NoError(err)
		s.Equal(expectedStatus, res.StatusCode)

		if updateDeployID && method == http.MethodDelete {
			var retCode int
			deployID, retCode, err = utils.StartDeployment(admclient, dpConfigName, "targeted", 10)
			s.Equal(retCode, 200)
			s.NoError(err)
			url = fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", deploymentRESTServerUrl, deployID)
		}

		s.T().Logf("%s method: %s (%d)\n", url, method, res.StatusCode)
	}
}

func (s *TestSuite) TestListDeploymentsMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments", deploymentRESTServerUrl)
	s.testMethod(url, methodStatusMap["listDeployments"], false)
}

func (s *TestSuite) TestListDeploymentsPerClusterMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/clusters/%s", deploymentRESTServerUrl, utils.TestClusterID)
	s.testMethod(url, methodStatusMap["listDeploymentsPerCluster"], false)
}

func (s *TestSuite) TestGetDeleteDeploymentMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s", deploymentRESTServerUrl, deployID)
	s.testMethod(url, methodStatusMap["getDeleteDeployment"], true)
}

func (s *TestSuite) TestGetDeploymentsStatusMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/summary/deployments_status", deploymentRESTServerUrl)
	s.testMethod(url, methodStatusMap["getDeploymentsStatus"], false)
}

func (s *TestSuite) TestListDeploymentClustersMethod() {
	url := fmt.Sprintf("%s/deployment.orchestrator.apis/v1/deployments/%s/clusters", deploymentRESTServerUrl, deployID)
	s.testMethod(url, methodStatusMap["listDeploymentClusters"], false)
}
