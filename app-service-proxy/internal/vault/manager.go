// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"time"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

const (
	secretPathPrefix = "app-service-proxy-agent-token" // #nosec G101
	TokenKey         = "token"
	TTLHoursKey      = "ttlHours"
)

func NewManager() Manager {
	return &manager{
		endpoint:       utils.GetSecretServiceEndpoint(),
		serviceAccount: utils.GetServiceAccount(),
		mount:          utils.GetSecretServiceMount(),
	}
}

type Manager interface {
	GetToken(ctx context.Context, clusterID string) (*Token, error)
	PutToken(ctx context.Context, clusterID string, tokenValue string, tokenTTLHours int) error
	DeleteToken(ctx context.Context, clusterID string) error
	GetHarborCred(ctx context.Context) (*HarborCred, error)
	GetGitRepoCred(ctx context.Context) (map[string]string, error)
}

type manager struct {
	endpoint       string
	serviceAccount string
	mount          string
}

type Token struct {
	Value       string
	UpdatedTime time.Time
	TTLHours    int
}

type HarborCred struct {
	Username string
	Password string
	CaCert   string
}

func (m *manager) login(ctx context.Context) (*vault.Client, error) {
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

func (m *manager) logout(ctx context.Context, client *vault.Client) error {
	err := client.Auth().Token().RevokeSelfWithContext(ctx, "")
	if err != nil {
		return errors.NewInternal("failed to revoke token: %v", err)
	}
	return nil
}

func (m *manager) getPath(clusterID string) string {
	return fmt.Sprintf("%s/%s", secretPathPrefix, clusterID)
}

func (m *manager) getVaultKV2Store(_ context.Context, client *vault.Client) (*vault.KVv2, error) {
	kvStore := client.KVv2(m.mount)
	if kvStore == nil {
		return nil, errors.NewNotFound("KV data structure for mount %s does not exist", m.mount)
	}

	return kvStore, nil
}

func (m *manager) GetHarborCred(ctx context.Context) (*HarborCred, error) {
	cred := &HarborCred{}

	client, err := m.login(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := m.logout(ctx, client); err != nil {
			logrus.Errorf("Error logging out from Vault: %v\n", err)
		}
	}()
	kvStore, err := m.getVaultKV2Store(ctx, client)
	if err != nil {
		return nil, err
	}

	path := utils.GetSecretServiceHarborServicePath()
	secret, err := kvStore.Get(ctx, path)
	if err != nil {
		return nil, errors.NewInvalid("failed to get secret from path %s - err %+v", path, err)
	}

	if username, ok := secret.Data[utils.GetSecretServiceHarborServiceKVKeyUsername()]; ok {
		cred.Username = username.(string)
	}

	if password, ok := secret.Data[utils.GetSecretServiceHarborServiceKVKeyPassword()]; ok {
		cred.Password = password.(string)
	}

	if caCert, ok := secret.Data[utils.GetSecretServiceHarborServiceKVKeyCert()]; ok {
		cred.CaCert = caCert.(string)
	}

	return cred, nil
}

func (m *manager) GetToken(ctx context.Context, clusterID string) (*Token, error) {
	var tokenValue string
	var tokenUpdatedTime time.Time
	var tokenTTLHours int

	client, err := m.login(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := m.logout(ctx, client); err != nil {
			logrus.Errorf("Error logging out from Vault: %v\n", err)
		}
	}()
	kvStore, err := m.getVaultKV2Store(ctx, client)
	if err != nil {
		return nil, err
	}

	path := m.getPath(clusterID)
	secret, err := kvStore.Get(ctx, path)
	if err != nil {
		return nil, errors.NewInvalid("failed to get secret from path %s - err %+v", path, err)
	}

	metadata, err := kvStore.GetMetadata(ctx, path)
	if err != nil {
		return nil, errors.NewInvalid("failed to get secret metadata from path %s - err %+v", path, err)
	}

	if _, ok := secret.Data[TokenKey]; !ok {
		return nil, errors.NewNotFound("failed to find token value from path %s", path)
	}

	switch v := secret.Data[TokenKey].(type) {

	case string:
		tokenValue = v
	default:
		return nil, errors.NewInvalid("value type in token should be string: received - %v", v)
	}

	if _, ok := secret.Data[TTLHoursKey]; !ok {
		return nil, errors.NewNotFound("failed to find token TTL Hours from path %s", path)
	}

	switch v := secret.Data[TTLHoursKey].(type) {
	case json.Number:
		tokenTTLHours, err = strconv.Atoi(string(v))
		if err != nil {
			return nil, errors.NewInvalid("failed to convert ttlHours type from json.Number to int")
		}
	default:
		return nil, errors.NewInvalid("ttlHours type in token should be json.Number: received - type: %v", reflect.TypeOf(v))
	}

	tokenUpdatedTime = metadata.UpdatedTime

	return &Token{
		Value:       tokenValue,
		UpdatedTime: tokenUpdatedTime,
		TTLHours:    tokenTTLHours,
	}, nil
}

func (m *manager) PutToken(ctx context.Context, clusterID string, tokenValue string, tokenTTLHours int) error {
	data := map[string]interface{}{
		TokenKey:    tokenValue,
		TTLHoursKey: tokenTTLHours,
	}

	client, err := m.login(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := m.logout(ctx, client); err != nil {
			logrus.Errorf("Error logging out from Vault: %v\n", err)
		}
	}()
	kvStore, err := m.getVaultKV2Store(ctx, client)
	if err != nil {
		return err
	}

	path := m.getPath(clusterID)
	_, err = kvStore.Put(ctx, path, data)
	if err != nil {
		return errors.NewInvalid("failed to put data %+v to path %s for token %v / ttl %d", data, path, tokenValue, tokenTTLHours)
	}

	return nil
}

func (m *manager) DeleteToken(ctx context.Context, clusterID string) error {
	client, err := m.login(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := m.logout(ctx, client); err != nil {
			logrus.Errorf("Error logging out from Vault: %v\n", err)
		}
	}()
	kvStore, err := m.getVaultKV2Store(ctx, client)
	if err != nil {
		return err
	}

	path := m.getPath(clusterID)
	err = kvStore.Delete(ctx, path)
	if err != nil {
		return errors.NewInvalid("failed to delete token for the path %s", path)
	}

	return nil
}

func (m *manager) GetGitRepoCred(ctx context.Context) (map[string]string, error) {

	data := make(map[string]string)

	client, err := m.login(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := m.logout(ctx, client); err != nil {
			logrus.Errorf("Error logging out from Vault: %v\n", err)
		}
	}()
	kvStore, err := m.getVaultKV2Store(ctx, client)
	if err != nil {
		return nil, err
	}

	path := utils.GetSecretServiceGitServicePath()
	secret, err := kvStore.Get(ctx, path)
	if err != nil {
		return nil, errors.NewInvalid("failed to get secret from path %s - err %+v", path, err)
	}

	if username, ok := secret.Data[utils.GetSecretServiceGitServiceKVKeyUsername()]; ok {
		data["username"] = username.(string)
	}

	if password, ok := secret.Data[utils.GetSecretServiceGitServiceKVKeyPassword()]; ok {
		data["password"] = password.(string)
	}

	return data, nil
}
