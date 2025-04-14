// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/test/utils"
)

func (s *TestSuite) TestAuthProjectIDDelete() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, s.token, "s.projectID")
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 403")
	s.Error(err)
	s.T().Logf("successfully handled invalid projectid to delete pod\n")
}
func (s *TestSuite) TestAuthJWTDelete() {
	armClient, err := utils.CreateArmClient(s.ResourceRESTServerUrl, "stoken", s.projectID)
	s.NoError(err)

	err = PodDelete(armClient, "namespace", "podname")
	s.Equal(err.Error(), "failed to delete pod: <nil>, status: 500")
	s.Error(err)
	s.T().Logf("successfully handled invalid JWT to delete pod\n")
}
