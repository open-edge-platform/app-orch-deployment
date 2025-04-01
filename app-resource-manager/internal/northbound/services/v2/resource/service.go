// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/southbound"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc"
)

var log = dazl.GetPackageLogger()

type Service struct {
	sbHandler southbound.Handler
	opaClient openpolicyagent.ClientWithResponsesInterface
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
	sbHandler southbound.Handler
	opaClient openpolicyagent.ClientWithResponsesInterface
}

func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		sbHandler: s.sbHandler,
		opaClient: s.opaClient,
	}
	resourceapiv2.RegisterAppWorkloadServiceServer(r, server)
	resourceapiv2.RegisterEndpointsServiceServer(r, server)
	resourceapiv2.RegisterPodServiceServer(r, server)
	resourceapiv2.RegisterVirtualMachineServiceServer(r, server)

}

func NewService(sbHandler southbound.Handler, opaClient openpolicyagent.ClientWithResponsesInterface) northbound.Service {
	return &Service{
		sbHandler: sbHandler,
		opaClient: opaClient,
	}
}
