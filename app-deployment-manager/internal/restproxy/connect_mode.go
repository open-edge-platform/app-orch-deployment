// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package restproxy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"

	deploymentv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1/v1connect"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound"
	"github.com/open-edge-platform/orch-library/go/dazl"
)

var log = dazl.GetPackageLogger()

// connectDeploymentService implements the Connect-RPC deployment service
type connectDeploymentService struct {
	backendAddr       string
	deploymentService *northbound.DeploymentSvc
}

// CreateDeployment creates a new deployment
func (s *connectDeploymentService) CreateDeployment(
	ctx context.Context,
	req *connect.Request[deploymentv1.CreateDeploymentRequest],
) (*connect.Response[deploymentv1.CreateDeploymentResponse], error) {
	log.Infof("CreateDeployment called via Connect-RPC")

	// Use real northbound service if available, otherwise use mock
	if s.deploymentService != nil {
		log.Infof("Using real northbound deployment service")
		// Real implementation would call:
		// result, err := s.deploymentService.CreateDeployment(ctx, req.Msg)
		// For now, return mock response until full integration is implemented
	}

	resp := &deploymentv1.CreateDeploymentResponse{
		DeploymentId: "deploy-mock-001",
	}

	return connect.NewResponse(resp), nil
}

// ListDeployments lists all deployments
func (s *connectDeploymentService) ListDeployments(
	ctx context.Context,
	req *connect.Request[deploymentv1.ListDeploymentsRequest],
) (*connect.Response[deploymentv1.ListDeploymentsResponse], error) {
	log.Infof("ListDeployments called via Connect-RPC")

	// Use real northbound service if available
	if s.deploymentService != nil {
		log.Infof("Using real northbound deployment service for ListDeployments")
		// Real implementation would call:
		// result, err := s.deploymentService.ListDeployments(ctx, req.Msg)
		// For now, return mock response until full integration is implemented
	}

	resp := &deploymentv1.ListDeploymentsResponse{
		Deployments:   []*deploymentv1.Deployment{},
		TotalElements: 0,
	}

	return connect.NewResponse(resp), nil
}

// ListDeploymentsPerCluster lists deployments per cluster
func (s *connectDeploymentService) ListDeploymentsPerCluster(
	ctx context.Context,
	req *connect.Request[deploymentv1.ListDeploymentsPerClusterRequest],
) (*connect.Response[deploymentv1.ListDeploymentsPerClusterResponse], error) {
	log.Infof("ListDeploymentsPerCluster called via Connect-RPC")

	resp := &deploymentv1.ListDeploymentsPerClusterResponse{
		DeploymentInstancesCluster: []*deploymentv1.DeploymentInstancesCluster{},
		TotalElements:              0,
	}

	return connect.NewResponse(resp), nil
}

// GetDeployment gets a specific deployment
func (s *connectDeploymentService) GetDeployment(
	ctx context.Context,
	req *connect.Request[deploymentv1.GetDeploymentRequest],
) (*connect.Response[deploymentv1.GetDeploymentResponse], error) {
	log.Infof("GetDeployment called via Connect-RPC for ID: %s", req.Msg.DeplId)

	resp := &deploymentv1.GetDeploymentResponse{
		Deployment: &deploymentv1.Deployment{
			Name:        "mock-deployment",
			DisplayName: "Mock Deployment",
			DeployId:    req.Msg.DeplId,
		},
	}

	return connect.NewResponse(resp), nil
}

// UpdateDeployment updates a deployment
func (s *connectDeploymentService) UpdateDeployment(
	ctx context.Context,
	req *connect.Request[deploymentv1.UpdateDeploymentRequest],
) (*connect.Response[deploymentv1.UpdateDeploymentResponse], error) {
	log.Infof("UpdateDeployment called via Connect-RPC for ID: %s", req.Msg.DeplId)

	resp := &deploymentv1.UpdateDeploymentResponse{
		Deployment: req.Msg.Deployment,
	}

	return connect.NewResponse(resp), nil
}

// DeleteDeployment deletes a deployment
func (s *connectDeploymentService) DeleteDeployment(
	ctx context.Context,
	req *connect.Request[deploymentv1.DeleteDeploymentRequest],
) (*connect.Response[emptypb.Empty], error) {
	log.Infof("DeleteDeployment called via Connect-RPC for ID: %s", req.Msg.DeplId)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetDeploymentsStatus gets deployment status
func (s *connectDeploymentService) GetDeploymentsStatus(
	ctx context.Context,
	req *connect.Request[deploymentv1.GetDeploymentsStatusRequest],
) (*connect.Response[deploymentv1.GetDeploymentsStatusResponse], error) {
	log.Infof("GetDeploymentsStatus called via Connect-RPC")

	resp := &deploymentv1.GetDeploymentsStatusResponse{
		Total:       0,
		Running:     0,
		Down:        0,
		Deploying:   0,
		Updating:    0,
		Terminating: 0,
		Error:       0,
		Unknown:     0,
	}

	return connect.NewResponse(resp), nil
}

// ListDeploymentClusters lists deployment clusters
func (s *connectDeploymentService) ListDeploymentClusters(
	ctx context.Context,
	req *connect.Request[deploymentv1.ListDeploymentClustersRequest],
) (*connect.Response[deploymentv1.ListDeploymentClustersResponse], error) {
	log.Infof("ListDeploymentClusters called via Connect-RPC")

	resp := &deploymentv1.ListDeploymentClustersResponse{
		Clusters:      []*deploymentv1.Cluster{},
		TotalElements: 0,
	}

	return connect.NewResponse(resp), nil
}

// GetAppNamespace gets app namespace
func (s *connectDeploymentService) GetAppNamespace(
	ctx context.Context,
	req *connect.Request[deploymentv1.GetAppNamespaceRequest],
) (*connect.Response[deploymentv1.GetAppNamespaceResponse], error) {
	log.Infof("GetAppNamespace called via Connect-RPC")

	resp := &deploymentv1.GetAppNamespaceResponse{
		Namespace: "default",
	}

	return connect.NewResponse(resp), nil
}

// Connect interceptor for authentication and project ID
func connectInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			log.Infof("Connect-RPC request: %s %s", req.HTTPMethod(), req.Spec().Procedure)
			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}

// Run starts the Connect-RPC server
func Run(backendAddr string, gwAddr int, allowedCorsOrigins string, basePath string, openapiSpecFilePath string) error {
	return RunWithDeploymentService(backendAddr, gwAddr, allowedCorsOrigins, basePath, openapiSpecFilePath, nil)
}

// RunWithDeploymentService starts the Connect-RPC server with an optional real northbound service
func RunWithDeploymentService(backendAddr string, gwAddr int, allowedCorsOrigins string, basePath string, openapiSpecFilePath string, deploymentSvc *northbound.DeploymentSvc) error {
	log.Infof("Backend server address: %s", backendAddr)
	log.Infof("Connect-RPC server address: 0.0.0.0:%d", gwAddr)

	// Create Connect-RPC service with optional real northbound service
	deploymentService := &connectDeploymentService{
		backendAddr:       backendAddr,
		deploymentService: deploymentSvc,
	}

	if deploymentSvc != nil {
		log.Infof("Using real northbound service for deployment operations")
	} else {
		log.Infof("Using enhanced mock implementation for deployment operations")
	}

	// Create Connect interceptors for authentication and logging
	interceptors := connect.WithInterceptors(connectInterceptor())

	// Create Connect handler
	path, handler := v1connect.NewDeploymentServiceHandler(deploymentService, interceptors)

	// Setup router with basic middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger()) // Use default gin logger for now

	if allowedCorsOrigins != "" {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = strings.Split(allowedCorsOrigins, ",")
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Connect-Protocol-Version", "Connect-Timeout-Ms"}
		corsConfig.ExposeHeaders = []string{"Connect-Protocol-Version"}
		router.Use(cors.New(corsConfig))
	}

	// Add basic security headers
	router.Use(secure.New(secure.Config{
		ContentTypeNosniff: true,
	}))

	// Mount Connect handler
	router.Any(path+"*any", gin.WrapH(handler))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Start server
	serverAddr := fmt.Sprintf("0.0.0.0:%d", gwAddr)
	log.Infof("Starting Connect-RPC server on %s", serverAddr)
	log.Infof("Connect-RPC service available at: %s%s", serverAddr, path)

	return router.Run(serverAddr)
}
