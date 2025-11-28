// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package fuzztests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/admclient"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type FuzzTestSuite struct {
	ctx        context.Context
	cancel     context.CancelFunc
	server     *server.Server
	addr       string
	wg         sync.WaitGroup
	httpClient *http.Client
}

func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

func setupDummyHttpServer(t testing.TB) (*http.Server, int) {
	port, err := getFreePort()
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html") // for HTML rewrites in transport
		w.Write([]byte("<html><body>Dummy HTTP Server</body></html>"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Warnf("Dummy HTTP server exited: %v", err)
			t.Error(err)
		}
	}()

	time.Sleep(1 * time.Second)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/app-service-proxy-test", port))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	logrus.Warnf("Dummy HTTP server started on port %d", port)

	return server, port
}

func setupFuzzTest(t testing.TB, enableAuth bool) *FuzzTestSuite {
	os.Setenv("ASP_LOG_LEVEL", "error")
	if enableAuth {
		os.Setenv("OIDC_SERVER_URL", "test.com")
		os.Setenv("OPA_ENABLED", "true")
	} else {
		os.Setenv("OIDC_SERVER_URL", "")
		os.Setenv("OPA_ENABLED", "false")
	}
	os.Setenv("OPA_PORT", "1234")
	os.Setenv("TOKEN_TTL_HOURS", "100")
	os.Setenv("RATE_LIMITER_QPS", "30")
	os.Setenv("RATE_LIMITER_BURST", "2000")

	s := &FuzzTestSuite{}
	s.addr = "127.0.0.1:8124"
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Set up dummy HTTP server first to get its port
	httpServer, httpPort := setupDummyHttpServer(t)
	os.Setenv("CCG_ADDRESS", fmt.Sprintf("localhost:%d", httpPort))

	os.Setenv("GIT_REPO_NAME", "mock-git-repo")
	os.Setenv("GIT_SERVER", "mock-git-server")
	os.Setenv("GIT_PROVIDER", "mock-git-provider")
	os.Setenv("PROXY_SERVER_URL", "wss://app-orch.kind.internal/app-service-proxy")
	os.Setenv("SECRET_SERVICE_ENABLED", "true")
	os.Setenv("AGENT_TARGET_NAMESPACE", "mock-app-namespace")
	os.Setenv("AUTH_TOKEN_SERVICE_ACCOUNT", "mock-service-account")
	os.Setenv("AUTH_TOKEN_EXPIRATION", "100")

	var err error
	s.server, err = server.NewServer(s.addr)
	require.NoError(t, err)

	// Initialize your server here with the necessary environment setup if needed
	admclient.NewClient = NewMockAdmNewClient

	// Start the server in a goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.server.Run()
		if err != nil {
			t.Error(err)
			s.cancel()
		}
		logrus.Warnf("ASP server exited: %v", err)
	}()

	// Close servers when the context is done
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.ctx.Done()
		logrus.Warnf("Shutting down dummy HTTP server")
		if err := httpServer.Shutdown(s.ctx); err != nil {
			t.Error(err)
		}
		logrus.Warnf("Shutting down ASP server")
		s.server.Close()
	}()

	// Give the server a moment to start up
	time.Sleep(1 * time.Second)

	// Send a request to the server
	resp, err := http.Get("http://" + s.addr + "/app-service-proxy-test")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Create a custom HTTP client with connection limits to prevent resource exhaustion
	// during long-running fuzz tests
	s.httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   true, // Don't keep connections alive during fuzzing
		},
		Timeout: 30 * time.Second,
	}

	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetReportCaller(true)

	return s
}

func FuzzRouter(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t, false)
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
		require.NoError(t, err)
		resp, err := s.httpClient.Do(req)
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
	s := setupFuzzTest(t, true)
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
		req.Header.Add("X-Forwarded-Proto", "https")
		resp, err := s.httpClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		err = resp.Body.Close()
		require.NoError(t, err)
	})
}

func FuzzProxyHeaderProto(f *testing.F) {
	t := &testing.T{}
	s := setupFuzzTest(t, true)
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
		req.Header.Add("X-Forwarded-Host", "app-service-proxy.kind.internal")
		req.Header.Add("X-Forwarded-Proto", escapeHeaderValue(seedData))
		req.Header.Add("X-Forwarded-Host", "app-service-proxy.kind.internal")
		req.Header.Add("X-Forwarded-Port", "8080")
		resp, err := s.httpClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
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
