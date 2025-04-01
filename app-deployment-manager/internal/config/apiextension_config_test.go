// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAuthType_String(t *testing.T) {
	a := AuthTypeMTLS
	assert.Equal(t, "mtls", a.String())
}

func TestEndpointScheme_String(t *testing.T) {
	e := EndpointSchemeHTTP
	assert.Equal(t, "http", e.String())
}

func TestEndpointScheme_GetPort(t *testing.T) {
	e := EndpointSchemeHTTP
	assert.Equal(t, EndpointSchemeHTTPPort, e.GetPort())
}

func TestSetAPIExtensionConfig(t *testing.T) {
	err := SetAPIExtensionConfig(&APIExtensionConfig{
		APIAgentNamespace: "orch-app",
	})

	cfg := GetAPIExtensionConfig()
	assert.Equal(t, "orch-app", cfg.APIAgentNamespace)
	assert.NoError(t, err)
}
