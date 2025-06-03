// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	admclient "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	armclient "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"

	"net/http"
)

func CreateArmClient(restServerURL, token, projectID string) (*armclient.ClientWithResponses, error) {
	armClient, err := armclient.NewClientWithResponses(restServerURL, armclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return armClient, err
}

func CreateAdmClient(restServerURL, token, projectID string) (*admclient.ClientWithResponses, error) {
	armClient, err := admclient.NewClientWithResponses(restServerURL, admclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		auth.AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return armClient, err
}
