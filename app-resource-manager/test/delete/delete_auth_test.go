// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

func (s *TestSuite) TestDeleteAuthProjectID() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "invalidprojectid")
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 403")
	s.Error(err)
	s.T().Logf("successfully handled invalid projectid to delete pod\n")
}

func (s *TestSuite) TestDeleteAuthJWT() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, auth.InvalidJWT, s.projectID)
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 401")
	s.Error(err)
	s.T().Logf("successfully handled invalid JWT to delete pod\n")
}
