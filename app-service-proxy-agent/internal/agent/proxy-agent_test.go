// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/token"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

var (
	testProxyServerURL = "http://127.0.0.1:58080"
	testAgentID        = "cluster-01234567"
	testTokenFileName  = "test-token"
	testTokenContent   = "test-token"
)

func TestNewServiceProxyAgent_FailedGetServerURL(t *testing.T) {
	agent, err := NewServiceProxyAgent()
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestNewServiceProxyAgent_FailedGetAgentID(t *testing.T) {
	t.Setenv("PROXY_SERVER_URL", testProxyServerURL)

	agent, err := NewServiceProxyAgent()
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestNewServiceProxyAgent_FailedGetTokenFilePath(t *testing.T) {
	t.Setenv("PROXY_SERVER_URL", testProxyServerURL)
	t.Setenv("AGENT_ID", testAgentID)
	origTokenFilePath := token.TokenFilePath
	token.TokenFilePath = ""
	defer func() {
		// need to be reverted back to original value for other tests
		token.TokenFilePath = origTokenFilePath
	}()

	agent, err := NewServiceProxyAgent()
	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestNewServiceProxyAgent(t *testing.T) {
	t.Setenv("PROXY_SERVER_URL", testProxyServerURL)
	t.Setenv("AGENT_ID", testAgentID)
	origTokenFilePath := token.TokenFilePath
	token.TokenFilePath = filepath.Join(t.TempDir(), testTokenFileName)
	defer func() {
		// need to be reverted back to original value for other tests
		token.TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(token.TokenFilePath, []byte(testTokenContent), 0600)
	assert.NoError(t, err)

	agent, err := NewServiceProxyAgent()
	assert.NoError(t, err)
	assert.NotNil(t, agent)
}

func TestServiceProxyAgent_Run_Failed(t *testing.T) {
	t.Setenv("PROXY_SERVER_URL", testProxyServerURL)
	t.Setenv("AGENT_ID", testAgentID)
	origTokenFilePath := token.TokenFilePath
	token.TokenFilePath = filepath.Join(t.TempDir(), testTokenFileName)
	defer func() {
		// need to be reverted back to original value for other tests
		token.TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(token.TokenFilePath, []byte(testTokenContent), 0600)
	assert.NoError(t, err)

	agent, err := NewServiceProxyAgent()
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	defer func() {
		if r := recover(); r == nil {
			t.Error("call agent.Run succeed - expected: failed")
		}
	}()

	agent.Run(context.TODO())
}

func TestServiceProxyAgent_Stop(t *testing.T) {
	t.Setenv("PROXY_SERVER_URL", testProxyServerURL)
	t.Setenv("AGENT_ID", testAgentID)
	origTokenFilePath := token.TokenFilePath
	token.TokenFilePath = filepath.Join(t.TempDir(), testTokenFileName)
	defer func() {
		// need to be reverted back to original value for other tests
		token.TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(token.TokenFilePath, []byte(testTokenContent), 0600)
	assert.NoError(t, err)

	agent, err := NewServiceProxyAgent()
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	agent.Stop()
	select {
	case <-agent.closed:
	default:
		t.Error("failed to close channel")
	}
}
