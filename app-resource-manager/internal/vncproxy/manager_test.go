// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package vncproxy

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestNewManager(t *testing.T) {
	cfg := Config{
		WSPort:     8080,
		ConfigPath: "../../test/configs/arm_config.yaml",
		FileBase:   "../../vnc-proxy-web-ui",
	}
	m := NewManager(cfg)
	if m == nil {
		t.Errorf("NewManager() returned nil")
	}

	type testtype struct {
		url     string
		mimeype string
		code    int
	}

	testUrls := []testtype{
		{"/test", "text/plain; charset=utf-8", 200},
		{"/?project=p1&app=a1&cluster=c1&vm=v1", "text/html", 200},
		{"/?project=&app=a1&cluster=c1&vm=v1", "text/plain; charset=utf-8", 400},
		{"/?project=p1", "text/plain; charset=utf-8", 404},
		{"/vnc-proxy-main.js", "application/javascript", 200},
		{"/vnc-proxy-styles.css", "text/css", 200},
		{"/keycloak.min.js", "application/javascript", 200},
		{"/rfb.js", "application/javascript", 200},
	}

	for _, url := range testUrls {
		rec := httptest.NewRecorder()

		m.router.ServeHTTP(rec, httptest.NewRequest("GET", url.url, nil))
		assert.Equal(t, url.code, rec.Code, url.url)
		assert.Equal(t, url.mimeype, rec.Header().Get("Content-Type"), url.url)
	}
}
