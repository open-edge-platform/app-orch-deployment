// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package restproxy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/dazl"
	ginlogger "github.com/open-edge-platform/orch-library/go/pkg/logging/gin"
	ginutils "github.com/open-edge-platform/orch-library/go/pkg/middleware/gin"
	openapiutils "github.com/open-edge-platform/orch-library/go/pkg/openapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var log = dazl.GetPackageLogger()

var allowedHeaders = map[string]struct{}{
	"x-request-id": {},
}

const ActiveProjectID = "ActiveProjectID"

// ErrorResponse represents a structured error response for REST API
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the detailed error information
type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// mapGrpcToHTTPStatus maps gRPC status codes to appropriate HTTP status codes
func mapGrpcToHTTPStatus(grpcCode codes.Code) int {
	switch grpcCode {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// createErrorResponse creates a structured error response from gRPC status
func createErrorResponse(s *status.Status) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    int(s.Code()),
			Message: s.Message(),
			Status:  s.Code().String(),
		},
	}
}

// writeErrorResponse writes the error response to the HTTP response writer
func writeErrorResponse(w http.ResponseWriter, marshaler runtime.Marshaler, httpStatus int, errorResponse ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	if buf, err := marshaler.Marshal(errorResponse); err == nil {
		_, _ = w.Write(buf)
	}
}

// handleGrpcError processes gRPC errors and writes appropriate HTTP responses
func handleGrpcError(w http.ResponseWriter, marshaler runtime.Marshaler, grpcStatus *status.Status) {
	httpStatus := mapGrpcToHTTPStatus(grpcStatus.Code())
	errorResponse := createErrorResponse(grpcStatus)
	writeErrorResponse(w, marshaler, httpStatus, errorResponse)
}

// errorHandler provides enhanced error handling for gRPC-Gateway responses
// This filters and formats error details for better REST API responses
func errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	// Check if this is a gRPC error
	if grpcStatus, ok := status.FromError(err); ok {
		handleGrpcError(w, marshaler, grpcStatus)
		return
	}

	// Fall back to default handler for non-gRPC errors
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

func isHeaderAllowed(s string) (string, bool) {
	// check if allowedHeaders contain the header
	if _, isAllowed := allowedHeaders[s]; isAllowed {
		// send uppercase header
		return strings.ToUpper(s), true
	}
	// if not in the allowed header, don't send the header
	return s, false
}

func Run(grpcAddr string, gwAddr int, allowedCorsOrigins string, basePath string, openapiSpecFilePath string) error {
	log.Infof("Serving gRPC-Gateway on port %d", gwAddr)

	gin.DefaultWriter = ginlogger.NewWriter(log)

	// creating mux for gRPC gateway with enhanced error handling
	gwmux := runtime.NewServeMux(
		// convert header in response(going from gateway) from metadata received.
		runtime.WithOutgoingHeaderMatcher(isHeaderAllowed),
		runtime.WithMetadata(func(_ context.Context, request *http.Request) metadata.MD {
			authHeader := request.Header.Get("Authorization")
			projectIDHeader := request.Header.Get(ActiveProjectID)
			// send all the headers received from the client
			md := metadata.Pairs("auth", authHeader, "activeprojectid", projectIDHeader)

			return md
		}),
		// handle 405 method not allowed
		runtime.WithRoutingErrorHandler(ginutils.HandleRoutingError),
		// Enhanced error handling for better REST API responses
		runtime.WithErrorHandler(errorHandler),
	)

	// Register DeploymentService
	err := deploymentpb.RegisterDeploymentServiceHandlerFromEndpoint(context.Background(), gwmux, grpcAddr, []grpc.DialOption{
		grpc.WithBlock(), // nolint:staticcheck
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		log.Fatalf("Failed to register Deployment Service Handler %v", err)
		return err
	}

	// Register ClusterService
	err = deploymentpb.RegisterClusterServiceHandlerFromEndpoint(context.Background(), gwmux, grpcAddr, []grpc.DialOption{
		grpc.WithBlock(), // nolint:staticcheck
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	if err != nil {
		log.Fatalf("Failed to register Cluster Service Handler %v", err)
		return err
	}

	gwServer := gin.New()

	// check if another method is allowed for the current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed' and HTTP status code 405
	// otherwise will return 'Page Not Found' and HTTP status code 404.
	gwServer.HandleMethodNotAllowed = true

	msgSizeLimitBytes, err := utils.GetMessageSizeLimit()
	if err != nil {
		log.Fatalf("Failed to get msg size limit %v", err)
		return err
	}

	// Restrict REST request size of body
	gwServer.Use(ginutils.MessageSizeLimiter(msgSizeLimitBytes))

	// Prevent MIME sniffing
	gwServer.Use(secure.New(secure.Config{ContentTypeNosniff: true}))

	// Set gin logger using dazl
	gwServer.Use(ginlogger.NewGinLogger(log))

	// Refuse overly-long, malformed, and non-printable characters
	gwServer.Use(ginutils.UnicodePrintableCharsChecker())

	gwServer.Use(ginutils.PathParamUnicodeCheckerMiddleware())

	// Set a value for trusted proxies
	err = gwServer.SetTrustedProxies(nil)
	if err != nil {
		log.Fatal(err)
		return err
	}

	spec, err := openapiutils.LoadOpenAPISpec(openapiSpecFilePath)
	if err != nil {
		log.Fatalf("Failed to read OpenAPI spec %v", err)
		return err
	}

	// Get all routes and methods described in schema
	allPaths := openapiutils.ExtractAllPaths(spec)

	allowedMethods := []string{}
	for verb := range allPaths {
		allowedMethods = append(allowedMethods, verb)
	}

	corsOrigins := strings.Split(allowedCorsOrigins, ",")
	if len(corsOrigins) > 1 {
		config := cors.DefaultConfig()
		config.AllowOrigins = corsOrigins
		gwServer.Use(cors.New(config))
	}

	gwServer.StaticFile(fmt.Sprintf("%sdeployment.orchestrator.apis/v1", basePath), openapiSpecFilePath)

	// Only register routes that match the specified methods
	gwServer.Group(fmt.Sprintf("%sdeployment.orchestrator.apis/v1/*{grpc_gateway}", basePath)).Match(allowedMethods, "", gin.WrapH(gwmux))

	// Enable liveness and readiness check
	gwServer.Handle("GET", "/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	return gwServer.Run(fmt.Sprintf(":%d", gwAddr))
}
