// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fuzztests

import (
	"context"
	"fmt"
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/restproxy"
	libnorthbound "github.com/open-edge-platform/orch-library/go/pkg/northbound"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

type FuzzTestSuiteRest struct {
	client *restClient.ClientWithResponses
	mock   *mocks.MockDeploymentServiceServer
	gwport uint16
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

	grpcPort := 8080
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	gwAddr, err := getFreePort()
	require.NoError(f, err)
	s.gwport = uint16(gwAddr)

	serverConfig := libnorthbound.NewInsecureServerConfig(int16(grpcPort))
	srv := libnorthbound.NewServer(serverConfig)
	srv.AddService(s.mock)
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

	// Wait until the gateway is ready. TODO: wait for gRPC server as well?
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

	f.Fuzz(func(t *testing.T, m, p, h, a string) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, m, fmt.Sprintf("http://localhost:%v", s.gwport), nil)
		if err != nil {
			return
		}
		req.URL.Path = p // bypass the URL parsing to allow malformed paths
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
