// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package gitclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codecommit/codecommitiface"
	vault "github.com/hashicorp/vault/api"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	manager "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/vault"
	"github.com/stretchr/testify/mock"
)

type mockCodeCommitClient struct {
	codecommitiface.CodeCommitAPI
	mock.Mock
}

func (m *mockCodeCommitClient) CreateRepository(input *codecommit.CreateRepositoryInput) (*codecommit.CreateRepositoryOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*codecommit.CreateRepositoryOutput), args.Error(1)
}

func (m *mockCodeCommitClient) DeleteRepository(input *codecommit.DeleteRepositoryInput) (*codecommit.DeleteRepositoryOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*codecommit.DeleteRepositoryOutput), args.Error(1)
}

func (m *mockCodeCommitClient) CreateCodeCommitClient() error {
	args := m.Called()
	return args.Error(0)
}

type mockManager struct {
	endpoint       string
	serviceAccount string
	mount          string
}

func (m *mockManager) Logout(ctx context.Context, client *vault.Client) error {
	return nil
}

func (m *mockManager) GetVaultClient(ctx context.Context) (*vault.Client, error) {
	return nil, nil
}

func (m *mockManager) GetKVSecret(ctx context.Context, client *vault.Client, path string) (*vault.KVSecret, error) {
	return nil, nil
}

func (m *mockManager) GetSecretValueString(ctx context.Context, client *vault.Client, path string, key string) (string, error) {
	switch path {
	case utils.GetSecretServiceAWSServicePath():
		switch key {
		case utils.GetSecretServiceAWSServiceKVKeyAccessKey():
			return "test-access-key", nil
		case utils.GetSecretServiceAWSServiceKVKeySecretAccessKey():
			return "test-secret-access-key", nil
		case utils.GetSecretServiceAWSServiceKVKeyRegion():
			return "test-region", nil
		case utils.GetSecretServiceAWSServiceKVKeySSHKey():
			return "test-ssh-key", nil
		}
	case utils.GetSecretServiceGitServicePath():
		switch key {
		case utils.GetSecretServiceGitServiceKVKeyUsername():
			return "username", nil
		case utils.GetSecretServiceGitServiceKVKeyPassword():
			return "password", nil
		}
	}

	return "", fmt.Errorf("mock err: path %s, key %s", path, key)
}

func newMockManager(endpoint, serviceAccount, mount string) manager.Manager {
	return &mockManager{
		endpoint:       endpoint,
		serviceAccount: serviceAccount,
		mount:          mount,
	}
}
