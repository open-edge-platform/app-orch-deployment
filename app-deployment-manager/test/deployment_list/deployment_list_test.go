// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment_list

import (
	"net/http"
	"testing"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
)

func ptr[T any](v T) *T {
	return &v
}

func (s *TestSuite) TestListDeploymentsWithPagination() {
	s.T().Parallel()
	testName := "ListDeploymentsWithPagination"
	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		deploymentReq := deploymentutils.StartDeploymentRequest{
			AdmClient:         s.AdmClient,
			DpPackageName:     app,
			DeploymentType:    deploymentutils.DeploymentTypeTargeted,
			DeploymentTimeout: deploymentutils.DeploymentTimeout,
			DeleteTimeout:     deploymentutils.DeleteTimeout,
			TestName:          testName,
		}
		_, code, err := deploymentutils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")
	}

	deps, code, err := deploymentutils.DeploymentsListWithParams(s.AdmClient, &restClient.DeploymentServiceListDeploymentsParams{
		PageSize: ptr(int32(1)),
		Offset:   ptr(int32(0)),
		OrderBy:  ptr("deployPackage"),
	})
	s.NoError(err, "Failed to list deployments with pagination")
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
	s.NotEmpty(deps, "Expected non-empty deployments list")
	s.Len(*deps, 1, "Expected exactly one deployment in the list")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}

func (s *TestSuite) TestListDeploymentsInvalidPaginationParameters() {
	s.T().Parallel()
	testCases := []struct {
		pageSize int32
		offset   int32
	}{
		{pageSize: -1, offset: 0},  // Invalid page size
		{pageSize: 0, offset: -1},  // Invalid offset
		{pageSize: 200, offset: 0}, // Page size exceeds maximum limit
		// TODO: test orderBy?
	}
	for _, tt := range testCases {
		s.T().Run("PageSize="+string(tt.pageSize)+"_Offset="+string(tt.offset), func(t *testing.T) {
			deps, code, err := deploymentutils.DeploymentsListWithParams(s.AdmClient, &restClient.DeploymentServiceListDeploymentsParams{
				PageSize: &tt.pageSize,
				Offset:   &tt.offset,
			})
			s.Error(err, "Failed to list deployments with pagination")
			s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
			s.NotNil(deps, "Expected non-nil deployments list")
			s.Len(*deps, 0, "Expected no deployment in the list")
		})
	}
}

func (s *TestSuite) TestListDeploymentsWithFilter() {
	s.T().Parallel()
	testName := "ListDeploymentsWithFilter"
	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		deploymentReq := deploymentutils.StartDeploymentRequest{
			AdmClient:         s.AdmClient,
			DpPackageName:     app,
			DeploymentType:    deploymentutils.DeploymentTypeTargeted,
			DeploymentTimeout: deploymentutils.DeploymentTimeout,
			DeleteTimeout:     deploymentutils.DeleteTimeout,
			TestName:          testName,
		}
		_, code, err := deploymentutils.StartDeployment(deploymentReq)
		s.Equal(http.StatusOK, code)
		s.NoError(err, "Failed to create '"+app+"-"+deploymentutils.DeploymentTypeTargeted+"' deployment")
	}
	displayName := deploymentutils.FormDisplayName(deploymentutils.AppWordpress, testName)
	deps, code, err := deploymentutils.DeploymentsListWithParams(s.AdmClient, &restClient.DeploymentServiceListDeploymentsParams{
		Filter: ptr("displayName=" + displayName),
	})
	s.NoError(err, "Failed to list deployments with filter")
	s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with filter")
	s.NotEmpty(deps, "Expected non-empty deployments list")
	s.Len(*deps, 1, "Expected exactly one deployment in the list")

	for _, app := range []string{deploymentutils.AppWordpress, deploymentutils.AppNginx} {
		displayName := deploymentutils.FormDisplayName(app, testName)
		err := deploymentutils.DeleteAndRetryUntilDeleted(s.AdmClient, displayName, deploymentutils.RetryCount, deploymentutils.DeleteTimeout)
		s.NoError(err)
	}
}
