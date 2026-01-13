// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package restproxy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	resourcev2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/northbound/services/v2/resource/mocks"
	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
)

type FuzzTestSuiteRest struct {
	epMock *mocks.MockEndpointsServiceServer
	awMock *mocks.MockAppWorkloadServiceServer
	pMock  *mocks.MockPodServiceServer
	vmMock *mocks.MockVirtualMachineServiceServer
	gwport uint16
	srv    *northbound.Server
}

var (
	allowedCorsOrigins string = ""
	basePath           string = ""
)

func getFreePort() (port int, err error) {
	// Try to find a free port in the safe range for int16 (8000-32767)
	// Use random starting point to avoid conflicts
	start := 8000 + rand.Intn(20000)
	for i := 0; i < 5000; i++ {
		port = start + i
		if port >= 32767 {
			port = 8000 + (port % 24767)
		}
		var a *net.TCPAddr
		if a, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port)); err != nil {
			continue
		}
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			l.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found after 5000 attempts")
}

func setupFuzzTestREST(f testing.TB) *FuzzTestSuiteRest {
	t := &testing.T{}
	s := &FuzzTestSuiteRest{}
	os.Setenv("MSG_SIZE_LIMIT", "1")

	ctrl := gomock.NewController(t)
	s.epMock = mocks.NewMockEndpointsServiceServer(ctrl)
	s.awMock = mocks.NewMockAppWorkloadServiceServer(ctrl)
	s.pMock = mocks.NewMockPodServiceServer(ctrl)
	s.vmMock = mocks.NewMockVirtualMachineServiceServer(ctrl)

	grpcPort, err := getFreePort()
	require.NoError(f, err)
	grpcAddr := fmt.Sprintf("localhost:%d", grpcPort)

	gwAddr, err := getFreePort()
	require.NoError(f, err)
	s.gwport = uint16(gwAddr)

	// Note: casting to int16 may overflow for ports > 32767, but this is acceptable for testing
	// as the underlying gRPC server will use the correct uint16 port value
	serverConfig := northbound.NewInsecureServerConfig(int16(grpcPort))
	s.srv = northbound.NewServer(serverConfig)
	s.srv.AddService(s.epMock)
	s.srv.AddService(s.awMock)
	s.srv.AddService(s.pMock)
	s.srv.AddService(s.vmMock)
	doneCh := make(chan error, 1)
	go func() {
		err := s.srv.Serve(func(started string) {
			f.Log("gRPC server started on port", started)
			close(doneCh)
		}, grpc.MaxRecvMsgSize(1*1024*1024))
		if err != nil {
			doneCh <- err
		}
	}()

	// Wait for gRPC server to start before proceeding
	select {
	case err := <-doneCh:
		if err != nil {
			require.NoError(f, err, "gRPC server failed to start")
		}
	case <-time.After(5 * time.Second):
		require.FailNow(f, "gRPC server did not start within 5 seconds")
	}

	f.Cleanup(func() {
		if s.srv != nil {
			s.srv.Stop()
		}
	})

	gwReady := make(chan error, 1)
	go func() {
		f.Logf("gRPC gateway starting on port %v", gwAddr)
		err := Run(gwAddr, grpcAddr, basePath, allowedCorsOrigins, "../../api/nbi/v2/spec/v2/openapi.yaml")
		if err != nil {
			gwReady <- err
		}
	}()

	// Wait until the gateway is ready
	require.Eventually(f, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", gwAddr))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false
		}
		return true
	}, time.Second*10, time.Millisecond*500)

	return s
}

func FuzzRESTRouter(f *testing.F) {
	s := setupFuzzTestREST(f)
	s.epMock.EXPECT().ListAppEndpoints(gomock.Any(), &resourcev2.ListAppEndpointsRequest{AppId: "foo123", ClusterId: "bar456"}).
		Return(&resourcev2.ListAppEndpointsResponse{AppEndpoints: []*resourcev2.AppEndpoint{{Id: "ep789"}}}, nil).AnyTimes()
	s.awMock.EXPECT().ListAppWorkloads(gomock.Any(), &resourcev2.ListAppWorkloadsRequest{AppId: "foo123", ClusterId: "bar456"}).
		Return(&resourcev2.ListAppWorkloadsResponse{AppWorkloads: []*resourcev2.AppWorkload{{Id: "aw789"}}}, nil).AnyTimes()
	s.pMock.EXPECT().DeletePod(gomock.Any(), &resourcev2.DeletePodRequest{ClusterId: "bar456", Namespace: "ns123", PodName: "pod789"}).
		Return(&resourcev2.DeletePodResponse{}, nil).AnyTimes()
	s.vmMock.EXPECT().GetVNC(gomock.Any(), &resourcev2.GetVNCRequest{AppId: "foo123", ClusterId: "bar456", VirtualMachineId: "vm789"}).
		Return(&resourcev2.GetVNCResponse{Address: "vnc-address"}, nil).AnyTimes()
	s.vmMock.EXPECT().StartVirtualMachine(gomock.Any(), &resourcev2.StartVirtualMachineRequest{AppId: "foo123", ClusterId: "bar456", VirtualMachineId: "vm789"}).
		Return(&resourcev2.StartVirtualMachineResponse{}, nil).AnyTimes()
	s.vmMock.EXPECT().StopVirtualMachine(gomock.Any(), &resourcev2.StopVirtualMachineRequest{AppId: "foo123", ClusterId: "bar456", VirtualMachineId: "vm789"}).
		Return(&resourcev2.StopVirtualMachineResponse{}, nil).AnyTimes()
	s.vmMock.EXPECT().RestartVirtualMachine(gomock.Any(), &resourcev2.RestartVirtualMachineRequest{AppId: "foo123", ClusterId: "bar456", VirtualMachineId: "vm789"}).
		Return(&resourcev2.RestartVirtualMachineResponse{}, nil).AnyTimes()

	f.Add("GET", "", "", "")
	f.Add("GET", "/", "Accept-Encoding", "gzip")
	f.Add("GET", "/", "Authorization", "Bearer token")
	f.Add("GET", "/healthz", "", "")
	f.Add("GET", "/test", "", "")
	// All actual paths from the openapi spec.
	f.Add("PUT", "/resource.orchestrator.apis/v2/workloads/pods/bar456/ns123/pod789/delete", "", "")
	f.Add("GET", "/resource.orchestrator.apis/v2/workloads/virtual-machines/foo123/bar456/vm789/vnc", "", "")
	f.Add("PUT", "/resource.orchestrator.apis/v2/workloads/virtual-machines/foo123/bar456/vm789/restart", "", "")
	f.Add("PUT", "/resource.orchestrator.apis/v2/workloads/virtual-machines/foo123/bar456/vm789/start", "", "")
	f.Add("PUT", "/resource.orchestrator.apis/v2/workloads/virtual-machines/foo123/bar456/vm789/stop", "", "")
	f.Add("GET", "/resource.orchestrator.apis/v2/workloads/foo123/bar456", "", "")
	f.Add("GET", "/resource.orchestrator.apis/v2/endpoints/foo123/bar456", "", "")

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
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	})
}
