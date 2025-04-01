// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"context"

	"github.com/atomix/atomix/api/errors"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

type NewManagerFunc func(endpoint, serviceAccount, mount string) Manager

// Allow overriding of vault.NewManager for gitclient unit testing
var NewManager NewManagerFunc = newManager

func newManager(endpoint, serviceAccount, mount string) Manager {
	return &manager{
		endpoint:       endpoint,
		serviceAccount: serviceAccount,
		mount:          mount,
	}
}

type Manager interface {
	GetVaultClient(ctx context.Context) (*vault.Client, error)
	GetKVSecret(ctx context.Context, client *vault.Client, path string) (*vault.KVSecret, error)
	GetSecretValueString(ctx context.Context, client *vault.Client, path string, key string) (string, error)
	Logout(ctx context.Context, client *vault.Client) error
}

type manager struct {
	endpoint       string
	serviceAccount string
	mount          string
}

func (m *manager) Logout(ctx context.Context, client *vault.Client) error {
	err := client.Auth().Token().RevokeSelfWithContext(ctx, "")
	if err != nil {
		return errors.NewInternal("failed to revoke token: %v", err)
	}
	return nil
}

func (m *manager) GetVaultClient(ctx context.Context) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = m.endpoint
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}

	auth, err := auth.NewKubernetesAuth(m.serviceAccount)
	if err != nil {
		return nil, err
	}
	authInfo, err := client.Auth().Login(ctx, auth)
	if err != nil {
		return nil, err
	}
	if authInfo == nil {
		return nil, errors.NewInvalid("kubernetes authorization failed for Vault service")
	}
	return client, nil
}

func (m *manager) GetKVSecret(ctx context.Context, client *vault.Client, path string) (*vault.KVSecret, error) {
	kvClient := client.KVv2(m.mount)
	if kvClient == nil {
		return nil, errors.NewInvalid("Vault secret KV data structure client is nil")
	}
	return kvClient.Get(ctx, path)
}

func (m *manager) GetSecretValueString(ctx context.Context, client *vault.Client, path string, key string) (string, error) {
	kvSecret, err := m.GetKVSecret(ctx, client, path)
	if err != nil {
		return "", err
	}

	if _, ok := kvSecret.Data[key]; !ok {
		return "", errors.NewNotFound("could not found value for key %s in path %s", key, path)
	}

	switch v := kvSecret.Data[key].(type) {
	case string:
		return v, nil
	default:
		return "", errors.NewInvalid("value for key %s in path %s is not string", key, path)
	}
}
