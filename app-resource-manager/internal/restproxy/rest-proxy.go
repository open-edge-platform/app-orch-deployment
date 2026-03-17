// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
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
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	envutils "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/env"
	"github.com/open-edge-platform/orch-library/go/dazl"
	orcherror "github.com/open-edge-platform/orch-library/go/pkg/errors"
	ginlogger "github.com/open-edge-platform/orch-library/go/pkg/logging/gin"
	ginmiddleware "github.com/open-edge-platform/orch-library/go/pkg/middleware/gin"
	"github.com/open-edge-platform/orch-library/go/pkg/middleware/projectcontext"
	openapiutils "github.com/open-edge-platform/orch-library/go/pkg/openapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const ActiveProjectID = "ActiveProjectID"

var log = dazl.GetPackageLogger()

var allowedHeaders = map[string]struct{}{
	"x-request-id": {},
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

// errorHandler provides enhanced error handling for gRPC-Gateway responses
func errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	if grpcStatus, ok := status.FromError(err); ok {
		typedErr := orcherror.FromStatus(grpcStatus)

		// Convert back to gRPC status to get the proper HTTP status code
		newStatus := orcherror.Status(typedErr)

		// Use the existing gRPC-Gateway error handler with the processed error
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, newStatus.Err())
		return
	}

	// Fall back to default handler for non-gRPC errors
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

// Run starts the ARM REST proxy server with the given configuration.
// Deprecated: use RunWithOptions instead.
func Run(restPort int, grpcEndpoint string, basePath string, allowedCorsOrigins string, openapiSpecFilePath string) error {
	return RunWithOptions(restPort, grpcEndpoint, basePath, allowedCorsOrigins, openapiSpecFilePath, "")
}

// RunWithOptions starts the ARM REST proxy server.
// nexusAPIURL is the URL of the Nexus API gateway used for project name resolution.
func RunWithOptions(restPort int, grpcEndpoint string, basePath string, allowedCorsOrigins string, openapiSpecFilePath string, nexusAPIURL string) error {
	gin.DefaultWriter = ginlogger.NewWriter(log)
	log.Infow("Starting REST proxy Server", dazl.Int("address", restPort))
	msgSizeLimitBytes, err := envutils.GetMessageSizeLimit()
	if err != nil {
		log.Fatalw("Failed to get msg size limit", dazl.Error(err))
		return err
	}

	// creating mux for gRPC gateway. This will multiplex or route request different gRPC service
	mux := runtime.NewServeMux(
		// convert header in response(going from gateway) from metadata received.
		runtime.WithOutgoingHeaderMatcher(isHeaderAllowed),
		runtime.WithMetadata(func(ctx context.Context, request *http.Request) metadata.MD {
			authHeader := request.Header.Get("Authorization")
			projectIDHeader := request.Header.Get(ActiveProjectID)
			// Resolve project ID from URL path or JWT, following EIM/infra-core pattern.
			// For new-style paths (/v1/projects/{name}/...), resolves name→UUID via Nexus.
			// For legacy paths, falls back to JWT extraction.
			projectUUID, err := projectcontext.ResolveAndValidateProjectID(
				ctx,
				request.URL.Path,
				authHeader,
				projectIDHeader,
				projectcontext.ProjectResolverConfig{
					ProjectServiceURL:     nexusAPIURL,
					ErrorOnMissingProject: false,
				},
			)
			if err != nil {
				log.Warnw("Failed to resolve project ID", dazl.Error(err))
			} else if projectUUID != "" {
				projectIDHeader = projectUUID
			}
			return metadata.Pairs("auth", authHeader, "activeprojectid", projectIDHeader)
		}),
		runtime.WithRoutingErrorHandler(ginmiddleware.HandleRoutingError),
		runtime.WithErrorHandler(errorHandler),
	)

	// Register V2 API Services
	err = resourceapiv2.RegisterEndpointsServiceHandlerFromEndpoint(context.Background(), mux, grpcEndpoint,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Fatalw("Failed to register V2 Endpoint Service  handler", dazl.Error(err))
		return err
	}

	err = resourceapiv2.RegisterAppWorkloadServiceHandlerFromEndpoint(context.Background(), mux, grpcEndpoint,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Fatalw("Failed to register V2 App Workload Service  handler", dazl.Error(err))
		return err
	}

	err = resourceapiv2.RegisterPodServiceHandlerFromEndpoint(context.Background(), mux, grpcEndpoint,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Fatalw("Failed to register V2 Pod Service  handler", dazl.Error(err))
		return err
	}

	err = resourceapiv2.RegisterVirtualMachineServiceHandlerFromEndpoint(context.Background(), mux, grpcEndpoint,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Fatalw("Failed to register V2 Virtual Machine Service  handler", dazl.Error(err))
		return err
	}

	server := gin.New()
	// check if another method is allowed for the current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed' and HTTP status code 405
	// otherwise will return 'Page Not Found' and HTTP status code 404.
	server.HandleMethodNotAllowed = true
	server.Handle("GET", "/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	server.Use(ginlogger.NewGinLogger(log))
	server.Use(secure.New(secure.Config{ContentTypeNosniff: true}))
	server.Use(ginmiddleware.PathParamUnicodeCheckerMiddleware())
	server.Use(ginmiddleware.MessageSizeLimiter(msgSizeLimitBytes))
	server.Use(ginmiddleware.UnicodePrintableCharsChecker())
	server.StaticFile(fmt.Sprintf("%sresource.orchestrator.apis/api/v2", basePath), openapiSpecFilePath)

	corsOrigins := strings.Split(allowedCorsOrigins, ",")
	if len(corsOrigins) > 1 {
		config := cors.DefaultConfig()
		config.AllowOrigins = corsOrigins
		server.Use(cors.New(config))
	}

	specV2, err := openapiutils.LoadOpenAPISpec(openapiSpecFilePath)
	if err != nil {
		log.Fatalw("Failed to load open API spec", dazl.Error(err))
		return err
	}

	allPathsV2 := openapiutils.ExtractAllPaths(specV2)

	var allowedMethodsV2 []string
	for verb := range allPathsV2 {
		allowedMethodsV2 = append(allowedMethodsV2, verb)
	}

	server.Group(fmt.Sprintf("%sresource.orchestrator.apis/v2/*{grpc_gateway}", basePath)).Match(allowedMethodsV2, "", gin.WrapH(mux))

	server.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Ok")
	})

	// start server
	return server.Run(fmt.Sprintf(":%d", restPort))
}
