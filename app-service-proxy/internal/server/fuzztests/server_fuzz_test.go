// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package fuzztests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/admclient"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/rbac"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FuzzTestSuite struct {
	ctx    context.Context
	cancel context.CancelFunc
	server *server.Server
	addr   string
}

type authenticateFunc func(req *http.Request) error
type authorizeFunc func(req *http.Request, projectID string) error

func setupFuzzTest(t testing.TB, authenticate authenticateFunc, authorize authorizeFunc) *FuzzTestSuite {
	os.Setenv("OIDC_SERVER_URL", "test.com")
	os.Setenv("OPA_ENABLED", "true")
	os.Setenv("OPA_PORT", "1234")
	os.Setenv("TOKEN_TTL_HOURS", "100")
	os.Setenv("RATE_LIMITER_QPS", "30")
	os.Setenv("RATE_LIMITER_BURST", "2000")
	os.Setenv("CCG_ADDRESS", "cluster-connect-gateway.orch-cluster.svc:8080")
	os.Setenv("GIT_REPO_NAME", "mock-git-repo")
	os.Setenv("GIT_SERVER", "mock-git-server")
	os.Setenv("GIT_PROVIDER", "mock-git-provider")
	os.Setenv("PROXY_SERVER_URL", "wss://app-orch.kind.internal/app-service-proxy")
	os.Setenv("SECRET_SERVICE_ENABLED", "true")
	os.Setenv("AGENT_TARGET_NAMESPACE", "mock-app-namespace")
	os.Setenv("AUTH_TOKEN_SERVICE_ACCOUNT", "mock-service-account")
	os.Setenv("AUTH_TOKEN_EXPIRATION", "100")
	auth.RenewTokenAuthorizer = func(req *http.Request, id string) (bool, error) { return true, nil }

	s := &FuzzTestSuite{}
	s.addr = "127.0.0.1:8124"
	s.ctx, s.cancel = context.WithCancel(context.Background())

	var err error
	rbac.AuthenticateFunc = authenticate
	rbac.AuthorizeFunc = authorize
	s.server, err = server.NewServer(s.addr)
	require.NoError(t, err)

	// Initialize your server here with the necessary environment setup if needed
	admclient.NewClient = NewMockAdmNewClient

	// Start the server in a goroutine
	go func() {
		select {
		case <-s.ctx.Done():
			return
		default:
			err := s.server.Run()
			if err != nil {
				t.Error(err)
				s.cancel()
			}
		}
	}()

	// Give the server a moment to start up
	time.Sleep(1 * time.Second)

	// Send a request to the server
	resp, err := http.Get("http://" + s.addr + "/test")
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	logrus.SetLevel(logrus.ErrorLevel)

	return s
}

func FuzzRouter(f *testing.F) {
	t := &testing.T{}
	authenticate := func(req *http.Request) error { return nil }
	authorize := func(req *http.Request, projectID string) error { return nil }
	s := setupFuzzTest(t, authenticate, authorize)
	s.server.Authenticate = authenticate
	s.server.Authorize = authorize
	os.Setenv("OIDC_SERVER_URL", "")
	os.Setenv("OPA_ENABLED", "false")
	defer s.cancel()
	f.Add("test")
	f.Fuzz(func(t *testing.T, seedData string) {
		url := fmt.Sprintf("http://%s/%s", s.addr, url2.PathEscape(seedData))
		req, err := http.NewRequest("GET", url, bytes.NewBufferString(""))
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-tokens", Value: "1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-token-0", Value: "123456"})
		req.Header.Add("X-Forwarded-Host", "app-service-proxy.kind.internal")
		req.Header.Add("X-Forwarded-Proto", "https")
		resp, err := http.DefaultClient.Do(req)
		fmt.Print("http err : ", err)
		require.NoError(t, err)
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusOK {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}

		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)
	})
}

func escapeHeaderValue(value string) string {
	var buffer bytes.Buffer
	for _, r := range value {
		if r < 32 || r >= 127 {
			buffer.WriteString(fmt.Sprintf("\\x%02X", r))
		} else {
			buffer.WriteRune(r)
		}
	}
	return buffer.String()
}

func FuzzProxyHeaderHost(f *testing.F) {
	t := &testing.T{}
	Authenticate := func(req *http.Request) error { return status.Errorf(codes.InvalidArgument, "Authenticate fail") }
	Authorize := func(req *http.Request, projectID string) error {
		return status.Errorf(codes.InvalidArgument, "Authorize fail")
	}
	s := setupFuzzTest(t, Authenticate, Authorize)
	defer s.cancel()
	f.Add("app-service-proxy.kind.internal")
	f.Fuzz(func(t *testing.T, seedData string) {
		url := fmt.Sprintf("http://%s/", s.addr)
		req, err := http.NewRequest("GET", url, bytes.NewBufferString(""))
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-tokens", Value: "1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-token-0", Value: "123456"})
		require.NoError(t, err)
		req.Header.Add("X-Forwarded-Host", escapeHeaderValue(seedData))
		//req.Header.Add("X-Forwarded-Proto", "https")
		s.server.Authenticate = Authenticate
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusOK {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}

		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)
	})
}

func FuzzProxyHeaderProto(f *testing.F) {
	t := &testing.T{}
	Authenticate := func(req *http.Request) error { return status.Errorf(codes.InvalidArgument, "Authenticate fail") }
	Authorize := func(req *http.Request, projectID string) error {
		return status.Errorf(codes.InvalidArgument, "Authorize fail")
	}
	s := setupFuzzTest(t, Authenticate, Authorize)
	defer s.cancel()
	f.Add("http")
	f.Fuzz(func(t *testing.T, seedData string) {
		url := fmt.Sprintf("http://%s/projects/project1/clusters/mock-cluster/api/v1/namespaces/mock-namespace/services/mock-service/proxy/", s.addr)
		req, err := http.NewRequest("GET", url, bytes.NewBufferString(""))
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-tokens", Value: "1"})
		req.AddCookie(&http.Cookie{Name: "app-service-proxy-token-0", Value: "123456"})
		require.NoError(t, err)
		req.Header.Add("X-Forwarded-Proto", escapeHeaderValue(seedData))
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		fmt.Print("http err : ", err)
		reqDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			fmt.Print(err)
		}

		// Print the request as a string
		fmt.Printf("%s\n", reqDump)
		//require.Equal(t, resp.StatusCode, http.StatusUnauthorized)
		if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusOK {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}

		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)
	})
}

type mockADMClient struct{}

func (c *mockADMClient) GetClusterToken(ctx context.Context, clusterId, namespace, serviceAccount string, expiration *int64) (string, error) {
	return "", nil
}

func (c *mockADMClient) GetClusterInfraName(ctx context.Context, clusterID,
	activeProjectID string) (string, error) {
	return "", nil
}

var NewMockAdmNewClient = func(opts ...grpc.DialOption) (admclient.ADMClient, error) {
	return &mockADMClient{}, nil
}
