// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package wsproxy

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const (
	testConfigValue = `
 appDeploymentManager:
    endpoint: "http://adm-api.orch-app.svc:8081"
 webSocketServer:
    protocol: "wss"
    hostName: "vnc.kind.internal"
    sessionLimitPerIP: 1
    sessionLimitPerAccount: 1
    readLimitByte: 0
    dlIdleTimeoutMin: 0
    ulIdleTimeoutMin: 0
    allowedOrigins:
      - https://vnc.kind.internal`
)

const (
	testConfigValueWrong = `
 appDeploymentManager:
    endpoint: "http://adm-api.orch-app.svc:8081"
 webSocketServer:
    protocol: "wss"
    hostName: "vnc.kind.internal"
    sessionLimitPerIP: -1
    sessionLimitPerAccount: -1
    readLimitByte: 0
    dlIdleTimeoutMin: 0
    ulIdleTimeoutMin: 0
    allowedOrigins:
      - https://vnc.kind.internal`
)

func TestCounterType_String_IP(t *testing.T) {
	counterType := CounterType(CounterTypeIP)
	assert.Equal(t, "CounterTypeIP", counterType.String())
}

func TestCounterType_String_Account(t *testing.T) {
	counterType := CounterType(CounterTypeAccount)
	assert.Equal(t, "CounterTypeAccount", counterType.String())
}

func TestNewCounter(t *testing.T) {
	c := NewCounter("", CounterTypeIP)
	assert.NotNil(t, c)
}

func TestCounter_IPCounter(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeIP)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.NoError(t, err)

	err = c.Increase("test")
	assert.Error(t, err)

	err = c.Decrease("test")
	assert.NoError(t, err)

	err = c.Decrease("test")
	assert.Error(t, err)
}

func TestCounter_UnknownCounter(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeUnknown)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.Error(t, err)

	err = c.Decrease("test")
	assert.Error(t, err)
}

func TestCounter_AccountCounter(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeAccount)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.NoError(t, err)

	err = c.Increase("test")
	assert.Error(t, err)

	err = c.Decrease("test")
	assert.NoError(t, err)

	err = c.Decrease("test")
	assert.Error(t, err)

	err = c.Decrease("test_foo")
	assert.Error(t, err)
}

func TestCounter_IPCounter_FailedGetMaxLimitPerAccount(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValueWrong), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeAccount)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.Error(t, err)
}

func TestCounter_IPCounter_FailedGetMaxLimitPerIP(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValueWrong), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeIP)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.Error(t, err)
}

func TestCounter_Print(t *testing.T) {
	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	defer os.RemoveAll(configFile.Name())
	assert.NoError(t, err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(t, err)

	c := NewCounter(configFile.Name(), CounterTypeAccount)
	assert.NotNil(t, c)

	err = c.Increase("test")
	assert.NoError(t, err)

	assert.Equal(t, "test: 1 / ", c.Print())
}
