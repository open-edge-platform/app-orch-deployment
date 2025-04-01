// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"context"
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
	"os"
	"testing"
)

type OPATestSuite struct {
	suite.Suite
	opaMock     *openpolicyagent.MockClientWithResponsesInterface
	opaClient   openpolicyagent.ClientWithResponsesInterface
	bearerKey   string
	bearerValue string
	md          metadata.MD
}

func (s *OPATestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	opaMock := openpolicyagent.NewMockClientWithResponsesInterface(mockController)
	s.opaMock = opaMock
	s.opaClient = opaMock
	s.bearerKey = "bearer"
	s.bearerValue = "test-value"
	s.md = metadata.Pairs(s.bearerKey, s.bearerValue)
}

func (s *OPATestSuite) SetupSuite() {

}

func (s *OPATestSuite) TearDownSuite() {
}

func (s *OPATestSuite) TearDownTest() {

}

func (s *OPATestSuite) TestIsOPAEnabled() {
	tests := []struct {
		name         string
		envVar       string
		expectedBool bool
	}{
		{
			name:         "opa enabled",
			envVar:       "true",
			expectedBool: true,
		},
		{
			name:         "opa disabled",
			envVar:       "false",
			expectedBool: false,
		},
		{
			name:         "invalid flag",
			envVar:       "not a bool",
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			err := os.Setenv(OPAEnabled, tt.envVar)
			assert.NoError(t, err)

			opaEnabledValue := IsOPAEnabled()
			assert.Equal(t, tt.expectedBool, opaEnabledValue)

			err = os.Unsetenv(OPAEnabled)
			assert.NoError(t, err)
		})
	}
}

func (s *OPATestSuite) TestNewOPAClient() {
	tests := []struct {
		name            string
		envVar          string
		opaPort         string
		oicdServerURL   string
		notNilOpaClient bool
		opaScheme       string
		opaHostname     string
	}{
		{
			name:            "opa enabled",
			envVar:          "true",
			opaPort:         "8181",
			oicdServerURL:   "https://keycloak.kind.com",
			notNilOpaClient: true,
			opaScheme:       "http",
			opaHostname:     "localhost",
		},
		{
			name:            "opa disabled",
			envVar:          "false",
			opaPort:         "8181",
			oicdServerURL:   "https://keycloak.kind.com",
			notNilOpaClient: false,
			opaScheme:       "http",
			opaHostname:     "localhost",
		},
		{
			name:            "invalid OPAEnabled flag",
			envVar:          "not a bool",
			oicdServerURL:   "https://keycloak.kind.com",
			opaPort:         "8181",
			notNilOpaClient: false,
			opaScheme:       "http",
			opaHostname:     "localhost",
		},
		{
			name:            "invalid opa port",
			envVar:          "true",
			oicdServerURL:   "https://keycloak.kind.com",
			opaPort:         "invalid_port",
			notNilOpaClient: false,
			opaScheme:       "http",
			opaHostname:     "localhost",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			err := os.Setenv(OPAEnabled, tt.envVar)
			assert.NoError(t, err)

			err = os.Setenv(OPAPort, tt.opaPort)
			assert.NoError(t, err)

			err = os.Setenv(OIDCServerURL, tt.oicdServerURL)
			assert.NoError(t, err)

			opaClient := NewOPAClient(tt.opaScheme, tt.opaHostname)
			if opaClient == nil {
				assert.False(t, tt.notNilOpaClient)
			} else {
				assert.True(t, tt.notNilOpaClient)
			}

			err = os.Unsetenv(OPAEnabled)
			assert.NoError(t, err)
		})
	}

}

func (s *OPATestSuite) TestIsAuthorized() {
	ctx := metadata.NewIncomingContext(context.Background(), s.md)
	result := openpolicyagent.OpaResponse_Result{}
	err := result.FromOpaResponseResult1(true)
	assert.NoError(s.T(), err)
	s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&openpolicyagent.PostV1DataPackageRuleResponse{
			JSON200: &openpolicyagent.OpaResponse{
				DecisionId: nil,
				Metrics:    nil,
				Result:     result,
			},
		}, nil,
	).AnyTimes()

	req := &resourceapiv2.StartVirtualMachineRequest{}
	err = IsAuthorized(ctx, req, s.opaClient)
	assert.NoError(s.T(), err)

}

func (s *OPATestSuite) TestIsAuthorizedError() {
	ctx := metadata.NewIncomingContext(context.Background(), s.md)
	s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		nil, errors.NewInvalid("opa internal error"),
	).AnyTimes()

	req := &resourceapiv2.StartVirtualMachineRequest{}
	err := IsAuthorized(ctx, req, s.opaClient)
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsInvalid(err))

}

func (s *OPATestSuite) TestIsAuthorizedOPANull() {
	ctx := metadata.NewIncomingContext(context.Background(), s.md)
	req := &resourceapiv2.StartVirtualMachineRequest{}
	err := IsAuthorized(ctx, req, nil)
	assert.Nil(s.T(), err)
}

func (s *OPATestSuite) TestIsAuthorizedNoMetadata() {
	req := &resourceapiv2.StartVirtualMachineRequest{}
	err := IsAuthorized(context.Background(), req, s.opaClient)
	assert.True(s.T(), errors.IsInvalid(err))
}

func TestOPA(t *testing.T) {
	suite.Run(t, new(OPATestSuite))
}
