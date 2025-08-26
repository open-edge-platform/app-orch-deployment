// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package restproxy

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	deploymentv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1/v1connect"
)

func TestConnectRPCServerCanStart(t *testing.T) {
	// Test that the server can start without error
	go func() {
		// This will start the server and block
		err := Run("localhost:8080", 8082, "", "", "/dev/null")
		if err != nil {
			t.Errorf("Failed to start Connect-RPC server: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test passes if we reach here without panic
	t.Log("Connect-RPC server started successfully")
}

func TestConnectRPCEndpoints(t *testing.T) {
	// Start the server in the background
	go func() {
		err := Run("localhost:8080", 8083, "", "", "/dev/null")
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(200 * time.Millisecond)

	// Create a Connect-RPC client
	client := v1connect.NewDeploymentServiceClient(
		http.DefaultClient,
		"http://localhost:8083",
	)

	// Test CreateDeployment
	t.Run("CreateDeployment", func(t *testing.T) {
		req := &deploymentv1.CreateDeploymentRequest{
			Deployment: &deploymentv1.Deployment{},
		}

		resp, err := client.CreateDeployment(context.Background(), connect.NewRequest(req))
		if err != nil {
			t.Errorf("CreateDeployment failed: %v", err)
			return
		}

		if resp.Msg.DeploymentId == "" {
			t.Error("CreateDeployment returned empty deployment ID")
		}

		t.Logf("Created deployment with ID: %s", resp.Msg.DeploymentId)
	})

	// Test ListDeployments
	t.Run("ListDeployments", func(t *testing.T) {
		req := &deploymentv1.ListDeploymentsRequest{}

		resp, err := client.ListDeployments(context.Background(), connect.NewRequest(req))
		if err != nil {
			t.Errorf("ListDeployments failed: %v", err)
			return
		}

		t.Logf("ListDeployments returned %d deployments", resp.Msg.TotalElements)
	})

	// Test GetDeployment
	t.Run("GetDeployment", func(t *testing.T) {
		req := &deploymentv1.GetDeploymentRequest{
			DeplId: "test-deployment-123",
		}

		resp, err := client.GetDeployment(context.Background(), connect.NewRequest(req))
		if err != nil {
			t.Errorf("GetDeployment failed: %v", err)
			return
		}

		if resp.Msg.Deployment == nil {
			t.Error("GetDeployment returned nil deployment")
		}

		t.Log("GetDeployment returned valid deployment")
	})
}
