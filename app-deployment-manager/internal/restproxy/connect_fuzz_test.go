// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package restproxy

import (
	"context"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"connectrpc.com/connect"
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound"
	"google.golang.org/protobuf/proto"
)

// FuzzConnectRPCCreateDeployment fuzzes the Connect-RPC CreateDeployment endpoint
func FuzzConnectRPCCreateDeployment(f *testing.F) {
	// Seed with a valid CreateDeploymentRequest
	seedReq := &deploymentpb.CreateDeploymentRequest{
		Deployment: &deploymentpb.Deployment{
			Name:        "test-deployment",
			AppName:     "test-app",
			AppVersion:  "1.0.0", 
			ProfileName: "default",
		},
	}

	seedData, err := proto.Marshal(seedReq)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create a fuzz consumer
		consumer := fuzz.NewConsumer(data)
		
		// Generate fuzzed request
		req := &deploymentpb.CreateDeploymentRequest{}
		err := consumer.GenerateStruct(req)
		if err != nil {
			return
		}

		// Create Connect-RPC service with minimal mock
		deploymentSvc := &northbound.DeploymentSvc{}
		connectSvc := &connectDeploymentService{deploymentService: deploymentSvc}

		// Create Connect-RPC request
		connectReq := connect.NewRequest(req)
		ctx := context.Background()

		// Call the method - should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Connect-RPC CreateDeployment panicked: %v", r)
			}
		}()

		_, _ = connectSvc.CreateDeployment(ctx, connectReq)
	})
}

// FuzzConnectRPCListDeployments fuzzes the Connect-RPC ListDeployments endpoint
func FuzzConnectRPCListDeployments(f *testing.F) {
	// Seed with a valid ListDeploymentsRequest
	seedReq := &deploymentpb.ListDeploymentsRequest{
		Labels: []string{"test=true"},
	}

	seedData, err := proto.Marshal(seedReq)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seedData)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.NewConsumer(data)
		
		req := &deploymentpb.ListDeploymentsRequest{}
		err := consumer.GenerateStruct(req)
		if err != nil {
			return
		}

		deploymentSvc := &northbound.DeploymentSvc{}
		connectSvc := &connectDeploymentService{deploymentService: deploymentSvc}

		connectReq := connect.NewRequest(req)
		ctx := context.Background()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Connect-RPC ListDeployments panicked: %v", r)
			}
		}()

		_, _ = connectSvc.ListDeployments(ctx, connectReq)
	})
}
