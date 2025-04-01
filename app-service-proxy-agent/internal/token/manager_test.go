// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testToken         = "this-is-test-token"
	testTTLHour       = 1
	testAgentID       = "cluster-01234567"
	testServerURL     = "wss://example.com"
	testTimeout       = 1 * time.Minute
	testWaitForAction = 10 * time.Second
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	assert.NotNil(t, mgr)
}

func TestManagerRetryFunc(t *testing.T) {
	mgr := manager{}

	err := mgr.retryFunc(func() error {
		return fmt.Errorf("test-err")
	}, 1*time.Millisecond)

	assert.Error(t, err)
}

func TestManagerGetToken_FailedReadFile(t *testing.T) {
	mgr := manager{}
	tk, err := mgr.getToken()
	assert.Error(t, err)
	assert.Equal(t, "", tk)
}

func TestManagerGetToken(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	tk, err := mgr.getToken()
	assert.NoError(t, err)
	assert.Equal(t, testToken, tk)
}

func TestManagerGetTTLHours_FailedReadFile(t *testing.T) {
	mgr := manager{}
	hour, err := mgr.getTTLHours()
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), hour)
}

func TestManagerGetTTLHours_FailedAtoI(t *testing.T) {
	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err := os.WriteFile(TTLHoursFilePath, []byte("error"), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	hour, err := mgr.getTTLHours()
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), hour)
}

func TestManagerGetTTLHours(t *testing.T) {
	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err := os.WriteFile(TTLHoursFilePath, []byte(fmt.Sprintf("%d", testTTLHour)), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	hour, err := mgr.getTTLHours()
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(testTTLHour)*time.Hour, hour)
}

func TestManagerGetUpdatedTime_FailedReadFile(t *testing.T) {
	mgr := manager{}
	_, err := mgr.getUpdatedTime()
	assert.Error(t, err)
}
func TestManagerGetUpdatedTime_FailedParsingTime(t *testing.T) {
	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err := os.WriteFile(UpdatedTimeFilePath, []byte("test"), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	_, err = mgr.getUpdatedTime()
	assert.Error(t, err)
}

func TestManagerGetUpdatedTime(t *testing.T) {
	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err := os.WriteFile(UpdatedTimeFilePath, []byte(time.Now().Format(time.DateTime)), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	_, err = mgr.getUpdatedTime()
	assert.NoError(t, err)
}

func TestManagerGetHTTPClient(t *testing.T) {
	mgr := manager{}
	c := mgr.getHTTPClient()
	assert.NotNil(t, c)
}

func TestManagerVerifyTokenRefresh_FailedGetToken(t *testing.T) {
	mgr := manager{}
	err := mgr.verifyTokenRefresh()
	assert.Error(t, err)
}

func TestManagerVerifyTokenRefresh_FailedGetUpdatedTime(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	mgr := manager{
		token: testToken,
	}
	err = mgr.verifyTokenRefresh()
	assert.Error(t, err)
}

func TestManagerVerifyTokenRefresh(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	err = mgr.verifyTokenRefresh()
	assert.Error(t, err)
}

func TestManagerVerifyTokenRefresh_FailedVerifyTokenWithTokenValue(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err = os.WriteFile(UpdatedTimeFilePath, []byte(time.Now().Format(time.DateTime)), 0600)
	assert.NoError(t, err)

	mgr := manager{}
	err = mgr.verifyTokenRefresh()
	assert.NoError(t, err)
}

func TestManagerTokenRefreshPrep_FailedGetToken(t *testing.T) {
	mgr := manager{}

	err := mgr.tokenRefreshPrep()
	assert.Error(t, err)
}

func TestManagerTokenRefreshPrep_FailedGetTTLHours(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	mgr := manager{}

	err = mgr.tokenRefreshPrep()
	assert.Error(t, err)
}

func TestManagerTokenRefreshPrep_FailedGetUpdatedTime(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err = os.WriteFile(TTLHoursFilePath, []byte(fmt.Sprintf("%d", testTTLHour)), 0600)
	assert.NoError(t, err)

	mgr := manager{}

	err = mgr.tokenRefreshPrep()
	assert.Error(t, err)
}

func TestManagerTokenRefreshPrep_FailedGetAgentID(t *testing.T) {
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err = os.WriteFile(TTLHoursFilePath, []byte(fmt.Sprintf("%d", testTTLHour)), 0600)
	assert.NoError(t, err)

	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err = os.WriteFile(UpdatedTimeFilePath, []byte(time.Now().Format(time.DateTime)), 0600)
	assert.NoError(t, err)

	mgr := manager{}

	err = mgr.tokenRefreshPrep()
	assert.Error(t, err)
}

func TestManagerTokenRefreshPrep_FailedGetProxyServerURL(t *testing.T) {
	t.Setenv("AGENT_ID", testAgentID)
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err = os.WriteFile(TTLHoursFilePath, []byte(fmt.Sprintf("%d", testTTLHour)), 0600)
	assert.NoError(t, err)

	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err = os.WriteFile(UpdatedTimeFilePath, []byte(time.Now().Format(time.DateTime)), 0600)
	assert.NoError(t, err)

	mgr := manager{}

	err = mgr.tokenRefreshPrep()
	assert.Error(t, err)
}

func TestManagerTokenRefreshPrep(t *testing.T) {
	t.Setenv("AGENT_ID", testAgentID)
	t.Setenv("PROXY_SERVER_URL", testServerURL)
	origTokenFilePath := TokenFilePath
	TokenFilePath = filepath.Join(t.TempDir(), "token")
	defer func() {
		TokenFilePath = origTokenFilePath
	}()

	err := os.WriteFile(TokenFilePath, []byte(testToken), 0600)
	assert.NoError(t, err)

	origTTLHoursFilePath := TTLHoursFilePath
	TTLHoursFilePath = filepath.Join(t.TempDir(), "ttlHours")
	defer func() {
		TTLHoursFilePath = origTTLHoursFilePath
	}()

	err = os.WriteFile(TTLHoursFilePath, []byte(fmt.Sprintf("%d", testTTLHour)), 0600)
	assert.NoError(t, err)

	origUpdatedTime := UpdatedTimeFilePath
	UpdatedTimeFilePath = filepath.Join(t.TempDir(), "updatedTime")
	defer func() {
		UpdatedTimeFilePath = origUpdatedTime
	}()

	err = os.WriteFile(UpdatedTimeFilePath, []byte(time.Now().Format(time.DateTime)), 0600)
	assert.NoError(t, err)

	mgr := manager{}

	err = mgr.tokenRefreshPrep()
	assert.NoError(t, err)
}

func TestManagerRequestTokenRefresh_FailedCreateHttpRequest(t *testing.T) {
	mgr := manager{
		serverURL: string([]byte{2}),
	}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh_FailedHTTPCall(t *testing.T) {
	mgr := manager{}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh_StatusUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
	}))

	mgr := manager{
		serverURL: ts.URL,
	}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh_StatusTooEarly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooEarly)
	}))

	mgr := manager{
		serverURL: ts.URL,
	}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh_StatusInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
	}))

	mgr := manager{
		serverURL: ts.URL,
	}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh_StatusUnsupportedMediaType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
	}))

	mgr := manager{
		serverURL: ts.URL,
	}
	err := mgr.requestTokenRefresh()
	assert.Error(t, err)
}

func TestManagerRequestTokenRefresh(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	mgr := manager{
		serverURL: ts.URL,
	}
	err := mgr.requestTokenRefresh()
	assert.NoError(t, err)
}
