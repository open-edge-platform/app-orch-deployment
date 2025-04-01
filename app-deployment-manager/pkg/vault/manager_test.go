// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/undefinedlabs/go-mpatch"
	"os"
	"reflect"
	"testing"
)

const (
	testEndpoint      = "http://test.endpoint.svc:8800"
	testServiceAccont = "test-orch-svc"
	testMount         = "test-secret"
)

func unpatchAll(list []*mpatch.Patch) error {
	for _, p := range list {
		err := p.Unpatch()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestNewManager(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)
}

func TestManager_GetVaultClient(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		authPatch, err := mpatch.PatchMethod(kubernetes.NewKubernetesAuth, func(roleName string, opts ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
			return nil, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		var authAPI *api.Auth
		loginPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(authAPI), "Login", func(a *api.Auth, ctx context.Context, authMethod api.AuthMethod) (*api.Secret, error) {
			return &api.Secret{}, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		return []*mpatch.Patch{authPatch, loginPatch}
	}
	pList := patch(ctrl)
	rootClient, err := mgr.GetVaultClient(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, rootClient)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetVaultClient_FailedNewClient(t *testing.T) {
	os.Setenv("VAULT_RATE_LIMIT", "dummy")
	defer os.Unsetenv("VAULT_RATE_LIMIT")
	mgr := NewManager("wrong:99999", testServiceAccont, testMount)
	assert.NotNil(t, mgr)
	rootClient, err := mgr.GetVaultClient(context.Background())
	assert.Error(t, err)
	assert.Nil(t, rootClient)
}

func TestManager_GetVaultClient_FailedAuth(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		authPatch, err := mpatch.PatchMethod(kubernetes.NewKubernetesAuth, func(roleName string, opts ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
			return nil, errors.NewUnknown("test error")
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{authPatch}
	}
	pList := patch(ctrl)
	rootClient, err := mgr.GetVaultClient(context.Background())
	assert.Error(t, err)
	assert.Nil(t, rootClient)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetVaultClient_FailedAuthInfo(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		authPatch, err := mpatch.PatchMethod(kubernetes.NewKubernetesAuth, func(roleName string, opts ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
			return nil, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		var authAPI *api.Auth
		loginPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(authAPI), "Login", func(a *api.Auth, ctx context.Context, authMethod api.AuthMethod) (*api.Secret, error) {
			return nil, errors.NewUnknown("test error")
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		return []*mpatch.Patch{authPatch, loginPatch}
	}
	pList := patch(ctrl)
	rootClient, err := mgr.GetVaultClient(context.Background())
	assert.Error(t, err)
	assert.Nil(t, rootClient)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetVaultClient_EmptyAuthInfo(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		authPatch, err := mpatch.PatchMethod(kubernetes.NewKubernetesAuth, func(roleName string, opts ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
			return nil, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		var authAPI *api.Auth
		loginPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(authAPI), "Login", func(a *api.Auth, ctx context.Context, authMethod api.AuthMethod) (*api.Secret, error) {
			return nil, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		return []*mpatch.Patch{authPatch, loginPatch}
	}
	pList := patch(ctrl)
	rootClient, err := mgr.GetVaultClient(context.Background())
	assert.Error(t, err)
	assert.Nil(t, rootClient)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetKVSecret(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		var client *api.Client
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return &api.KVv2{}
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		var kvClient *api.KVv2
		secretPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(kvClient), "Get", func(k *api.KVv2, ctx context.Context, secretPath string) (*api.KVSecret, error) {
			return &api.KVSecret{}, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch, secretPatch}
	}
	pList := patch(ctrl)
	secret, err := mgr.GetKVSecret(context.Background(), &api.Client{}, "")
	assert.NoError(t, err)
	assert.NotNil(t, secret)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetKVSecret_FailedGetKVClient(t *testing.T) {
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := &api.Client{}

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch}
	}
	pList := patch(ctrl)
	rootClient, err := mgr.GetKVSecret(context.Background(), client, "")
	assert.Error(t, err)
	assert.Nil(t, rootClient)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetSecretValueString(t *testing.T) {
	expectedValue := "testValue"
	expectedKey := "testKey"
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		var client *api.Client
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return &api.KVv2{}
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		var kvClient *api.KVv2
		secretPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(kvClient), "Get", func(k *api.KVv2, ctx context.Context, secretPath string) (*api.KVSecret, error) {
			return &api.KVSecret{
				Data: map[string]interface{}{expectedKey: expectedValue},
			}, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch, secretPatch}
	}
	pList := patch(ctrl)
	str, err := mgr.GetSecretValueString(context.Background(), nil, "", expectedKey)

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, str)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetSecretValueString_NilKVClient(t *testing.T) {
	expectedKey := "testKey"
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		var client *api.Client
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch}
	}
	pList := patch(ctrl)
	str, err := mgr.GetSecretValueString(context.Background(), nil, "", expectedKey)

	assert.Error(t, err)
	assert.Equal(t, "", str)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetSecretValueString_FailedRetrieveValue(t *testing.T) {
	expectedValue := "testValue"
	expectedKey := "testKey"
	wrongKey := "wrongKey"
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		var client *api.Client
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return &api.KVv2{}
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		var kvClient *api.KVv2
		secretPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(kvClient), "Get", func(k *api.KVv2, ctx context.Context, secretPath string) (*api.KVSecret, error) {
			return &api.KVSecret{
				Data: map[string]interface{}{expectedKey: expectedValue},
			}, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch, secretPatch}
	}
	pList := patch(ctrl)
	str, err := mgr.GetSecretValueString(context.Background(), nil, "", wrongKey)

	assert.Error(t, err)
	assert.Equal(t, "", str)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_GetSecretValueString_WrongValueType(t *testing.T) {
	expectedKey := "testKey"
	mgr := NewManager(testEndpoint, testServiceAccont, testMount)
	assert.NotNil(t, mgr)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		var client *api.Client
		kvv2Patch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(client), "KVv2", func(c *api.Client, mountPath string) *api.KVv2 {
			fmt.Printf("%+v\n", "test")
			return &api.KVv2{}
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		var kvClient *api.KVv2
		secretPatch, err := mpatch.PatchInstanceMethodByName(reflect.TypeOf(kvClient), "Get", func(k *api.KVv2, ctx context.Context, secretPath string) (*api.KVSecret, error) {
			return &api.KVSecret{
				Data: map[string]interface{}{expectedKey: 1},
			}, nil
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}
		return []*mpatch.Patch{kvv2Patch, secretPatch}
	}
	pList := patch(ctrl)
	str, err := mgr.GetSecretValueString(context.Background(), nil, "", expectedKey)

	assert.Error(t, err)
	assert.Equal(t, "", str)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}
