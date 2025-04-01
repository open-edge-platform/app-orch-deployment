// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"github.com/stretchr/testify/assert"
	"os"
	"runtime"
	"testing"
)

const (
	testConfigValue = `
 appDeploymentManager:
    endpoint: "http://adm-api.orch-app.svc:8081"
 webSocketServer:
    protocol: "wss"
    hostName: "vnc.kind.internal"
    sessionLimitPerIP: 0
    sessionLimitPerAccount: 0
    readLimitByte: 0
    dlIdleTimeoutMin: 0
    ulIdleTimeoutMin: 0
    allowedOrigins:
      - https://vnc.kind.internal`
)

func TestGetConfigModel(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(t, err)

	cfg, err := GetConfigModel(configFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

}

func TestGetConfigModel_WrongPath(t *testing.T) {
	cfg, err := GetConfigModel("")
	assert.Nil(t, cfg)
	assert.Error(t, err)
}

func TestGetConfigModel_UnmarshalFailed(t *testing.T) {
	p, filename, l, ok := runtime.Caller(0)
	cfg, err := GetConfigModel(filename)
	assert.NotNil(t, p)
	assert.NotNil(t, l)
	assert.NotNil(t, ok)
	assert.Nil(t, cfg)
	assert.Error(t, err)
}

func TestGetConfigModel_ValidationFailed(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)

	cfg, err := GetConfigModel(configFile.Name())
	assert.Nil(t, cfg)
	assert.Error(t, err)
}
