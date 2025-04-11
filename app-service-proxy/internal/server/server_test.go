// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	"net/http"
	"net/http/httptest"

	//"net/http/httptest"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/admclient"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/auth"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/vault"
)

var _ = Describe("Server", func() {

	var (
		testServer *Server
		recorder   *httptest.ResponseRecorder
		request    *http.Request
		err        error
		addr       string
	)

	BeforeEach(func() {
		os.Setenv("OIDC_SERVER_URL", "test.com")
		os.Setenv("OPA_ENABLED", "true")
		os.Setenv("OPA_PORT", "1234")
		os.Setenv("RATE_LIMITER_QPS", "30")
		os.Setenv("RATE_LIMITER_BURST", "2000")
		os.Setenv("TOKEN_TTL_HOURS", "100")
		os.Setenv("CCG_ADDRESS", "localhost:8085")
		os.Setenv("GIT_REPO_NAME", "mock-git-repo")
		os.Setenv("GIT_SERVER", "mock-git-server")
		os.Setenv("GIT_PROVIDER", "mock-git-provider")
		os.Setenv("PROXY_SERVER_URL", "wss://app-orch.kind.internal/app-service-proxy")
		os.Setenv("SECRET_SERVICE_ENABLED", "true")
		os.Setenv("ASP_LOG_LEVEL", "debug")
		auth.RenewTokenAuthorizer = func(req *http.Request, id string) (bool, error) { return true, nil }
		addr = "127.0.0.1:8123"
	})

	Describe("Server Run", func() {
		Context("When a request is authenticated and authorized", func() {
			It("It should run", func() {
				// Initialize your server here with the necessary environment setup if needed
				testServer, err = NewServer(addr)
				Expect(err).NotTo(HaveOccurred())

				admclient.NewClient = NewMockAdmNewClient

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				// Start the server in a goroutine
				go func() {
					select {
					case <-ctx.Done():
						return
					default:
						err := testServer.Run()
						Expect(err).NotTo(HaveOccurred())
					}
				}()

				// Give the server a moment to start up
				time.Sleep(1 * time.Second)

				// Send a request to the server
				resp, err := http.Get("http://" + addr + "/app-service-proxy-test")
				Expect(err).NotTo(HaveOccurred())

				// Check that the response status code is 200 OK
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("New Server", func() {
		Context("When a server is created with CCG_ADDRESS not set", func() {
			It("Should not be created", func() {
				os.Setenv("CCG_ADDRESS", "")
				testServer, err = NewServer(addr)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("New Server", func() {
		Context("When a server is created with CATTLE_TUNNEL_DATA_DEBUG set to true", func() {
			It("Should not be created", func() {
				os.Setenv("CATTLE_TUNNEL_DATA_DEBUG", "true")
				testServer, err = NewServer(addr)
				Expect(testServer).ToNot(Equal(nil))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("New Server", func() {
		Context("When a server is created", func() {
			It("Should create server successfully", func() {
				testServer, err = NewServer(addr)
				Expect(testServer).ToNot(Equal(nil))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Authentication and Authorization", func() {
		Context("When a request is not authenticated and authorized", func() {
			It("Should process the request successfully", func() {
				testServer.authenticate = func(req *http.Request) error { return nil }
				testServer.authorize = func(req *http.Request, projectID string) error { return nil }
				testServer.vaultManager = &mockVaultManager{}
				testServer.admClient = &mockADMClient{}
			})
		})
	})

	Describe("Authentication and Authorization", func() {
		Context("When a request is authenticated and authorized", func() {
			It("Should process the request successfully", func() {
				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy-test", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("API extension", func() {
		Context("When a client makes a request to an api extension", func() {
			It("Should process the request successfully", func() {
				gitclient.NewGitClient = func(repoName string) (gitclient.Repository, error) { return &mockGitClient{}, nil }
				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy/renew/mock-cluster", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("Service link", func() {
		Context("When a authorized client makes a request to a service link", func() {
			It("Should process the request successfully", func() {
				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/projects/project1/clusters/mock-cluster/api/v1/namespaces/mock-namespace/services/mock-service/proxy/", bytes.NewBufferString(""))
				request.Header.Add("X-Forwarded-Host", "app-service-proxy.kind.internal")
				request.Header.Add("X-Forwarded-Proto", "http")
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "mock-cluster"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "mock-namespace"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "mock-service:80"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-tokens", Value: "1"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-token-0", Value: "123456"})
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadGateway))
			})
		})
	})

	Describe("Service link", func() {
		Context("When a authorized client makes a request to a service link", func() {
			It("Should process the request successfully", func() {
				testServer.authenticate = func(req *http.Request) error { return fmt.Errorf("unauthorized") }
				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/project/project1/cluster/mock-cluster/api/v1/namespace/mock-namespace/service/mock-service/proxy/", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				request.Header.Add("X-Forwarded-Host", "app-service-proxy.kind.internal")
				request.Header.Add("X-Forwarded-Proto", "http")
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "mock-cluster"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "mock-namespace"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "mock-service:80"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-tokens", Value: "1"})
				request.AddCookie(&http.Cookie{Name: "app-service-proxy-token-0", Value: "123456"})
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				testServer.authenticate = func(req *http.Request) error { return nil }
			})
		})
	})

	Describe("Service link", func() {
		Context("When a authorized client makes a request to a service link", func() {
			It("Should serve the web-login page", func() {
				// Change working directory so that web-login files can be found
				wd, err := os.Getwd()
				Expect(err).NotTo(HaveOccurred())
				err = os.Chdir("../../")
				Expect(err).NotTo(HaveOccurred())
				defer os.Chdir(wd)

				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy-index.html?project=test&cluster=testcluster&namespace=testnamespace&service=https:testservice:80", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))

				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy-index.html", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest)) // Missing query parameters

				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy-main.js", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))

				recorder = httptest.NewRecorder()
				request, err = http.NewRequest("GET", "http://127.0.0.1:8123/app-service-proxy-styles.css", bytes.NewBufferString(""))
				Expect(err).NotTo(HaveOccurred())
				testServer.router.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("doProxy Functionality", func() {
		Context("When URL parsing fails", func() {
			It("Should respond with a 500 status code", func() {
				recorder = httptest.NewRecorder()
				request, _ := http.NewRequest("GET", "http://127.0.0.1:8123/anything", nil)
				// Intentionally setting headers and not setting cookies to cause URL parsing to fail
				request.Header.Add("X-Forwarded-Host", "")
				request.Header.Add("X-Forwarded-Proto", "")

				testServer.ServicesProxy(recorder, request)

				Expect(recorder.Code).To(Equal(http.StatusFound))
				Expect(recorder.Header().Get("Location")).To(Equal("/app-service-proxy-index.html"))
			})
		})
		// Add more contexts for other scenarios...
	})
})

// Mock ADM Client
type mockADMClient struct{}

func (c *mockADMClient) GetClusterToken(ctx context.Context, clusterId, namespace, serviceAccount string, expiration *int64) (string, error) {
	return "", nil
}

func (c *mockADMClient) GetClusterInfraName(ctx context.Context, clusterID,
	activeProjectID string) (string,
	error) {
	return "", nil
}

var NewMockAdmNewClient = func(opts ...grpc.DialOption) (admclient.ADMClient, error) {
	return &mockADMClient{}, nil
}

// Mock Vault Manager
type mockVaultManager struct{}

func (v *mockVaultManager) GetToken(ctx context.Context, clusterID string) (*vault.Token, error) {
	return &vault.Token{
		Value:       "mock-token",
		UpdatedTime: time.Time{},
		TTLHours:    0,
	}, nil
}
func (v *mockVaultManager) PutToken(ctx context.Context, clusterID string, tokenValue string, tokenTTLHours int) error {
	return nil
}
func (v *mockVaultManager) DeleteToken(ctx context.Context, clusterID string) error {
	return nil
}
func (v *mockVaultManager) GetHarborCred(ctx context.Context) (*vault.HarborCred, error) {
	return nil, nil
}

func (v *mockVaultManager) GetGitRepoCred(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

// Mock Git Client
type mockGitClient struct{}

func (g *mockGitClient) ExistsOnRemote() (bool, error)   { return true, nil }
func (g *mockGitClient) Initialize(basedir string) error { return nil }
func (g *mockGitClient) Clone(basedir string) error      { return nil }
func (g *mockGitClient) CommitFiles() error              { return nil }
func (g *mockGitClient) PushToRemote() error             { return nil }
func (g *mockGitClient) Delete() error                   { return nil }
func (g *mockGitClient) CreateCodeCommitClient() error   { return nil }
func (g *mockGitClient) CreateRepo() error               { return nil }
func (g *mockGitClient) DeleteRepo() error               { return nil }
