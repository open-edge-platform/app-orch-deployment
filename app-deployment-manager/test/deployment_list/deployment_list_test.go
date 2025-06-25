// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment_list

import (
	"net/http"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	deploymentutils "github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/deployment"
)

func ptr[T any](v T) *T {
	return &v
}

func (s *TestSuite) TestListDeploymentsWithPagination() {
	// s.T().Parallel()
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

	testCases := []struct {
		pageSize int32
		offset   int32
	}{
		{pageSize: 1, offset: 0},  // First page with one deployment
		{pageSize: 2, offset: 0},  // First page with two deployments
		{pageSize: 1, offset: 1},  // Second page with one deployment
		{pageSize: 2, offset: 1},  // Second page with two deployments
		{pageSize: 1, offset: 50}, // Tenth page (should be empty)
		{pageSize: 2, offset: 50}, // Tenth page with two deployments (should be empty)
	}
	for _, tt := range testCases {
		deps, code, err := deploymentutils.DeploymentsListWithParams(s.AdmClient, &restClient.DeploymentServiceListDeploymentsParams{
			PageSize: &tt.pageSize,
			Offset:   &tt.offset,
		})
		s.NoError(err, "Failed to list deployments with pagination")
		s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
		if tt.pageSize == 1 && tt.offset < 2 {
			s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
			s.NotEmpty(deps, "Expected non-empty deployments list")
			s.Len(*deps, 1, "Expected exactly one deployment in the list")
		} else if tt.pageSize == 2 && tt.offset < 2 {
			s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
			s.NotEmpty(deps, "Expected non-empty deployments list")
			s.Len(*deps, 2, "Expected exactly two deployments in the list")
		} else {
			s.Equal(http.StatusOK, code, "Expected HTTP status 200 for listing deployments with pagination")
			s.Empty(deps, "Expected empty deployments list for page size and offset combination")
		}
	}

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
		labels   *[]string
	}{
		{pageSize: -1, offset: 0},                                  // Invalid page size
		{pageSize: 0, offset: -1},                                  // Invalid offset
		{pageSize: 200, offset: 0},                                 // Page size exceeds maximum limit
		{pageSize: 0, offset: 0, labels: &[]string{"tester=foo "}}, // Invalid whitespace in label
		{pageSize: 0, offset: 0, labels: &[]string{"tes?er=foo"}},  // Invalid non-alphanumeric in label
		{pageSize: 0, offset: 0, labels: &[]string{"tesTer=foo"}},  // Invalid uppercase in label
		// TODO: test orderBy?
	}
	for _, tt := range testCases {
		deps, code, err := deploymentutils.DeploymentsListWithParams(s.AdmClient, &restClient.DeploymentServiceListDeploymentsParams{
			PageSize: &tt.pageSize,
			Offset:   &tt.offset,
			Labels:   tt.labels,
		})
		s.NoErrorf(err, "Pagination parameters %v", tt)
		s.Equalf(http.StatusBadRequest, code, "Pagination parameters %v", tt)
		s.NotNilf(deps, "Pagination parameters %v", tt)
		s.Lenf(*deps, 0, "Pagination parameters %v", tt)
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
