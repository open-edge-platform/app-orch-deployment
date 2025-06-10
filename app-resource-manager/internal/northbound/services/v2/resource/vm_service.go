// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"

	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

func (s *Server) GetVNC(ctx context.Context, req *resourceapiv2.GetVNCRequest) (*resourceapiv2.GetVNCResponse, error) {
	log.Infow("Received get VNC console request", dazl.Stringer("request", req))
	if req == nil || req.AppId == "" || req.ClusterId == "" || req.VirtualMachineId == "" {
		errMsg := "Request is not valid; app ID or cluster ID or VM ID is empty"
		log.Warnw(errMsg, dazl.Stringer("request", req))
		return nil, errors.Status(errors.NewInvalid(errMsg)).Err()
	}

	err := req.ValidateAll()
	if err != nil {
		log.Warnw("Request validation is failed", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	// Validate ActiveProjectID is present and valid
	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Warnw("ActiveProjectID validation failed", dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if err := opa.IsAuthorized(ctx, req, s.opaClient); err != nil {
		log.Warnw("Access denied by OPA rules",
			dazl.String("AppID", req.AppId),
			dazl.String("ProjectID", activeProjectID),
			dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}

	vncAddress, err := s.sbHandler.AccessVMWithVNC(ctx, req.AppId, req.ClusterId, req.VirtualMachineId)
	if err != nil {
		log.Warnw("Failed to get VNC console address", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}
	logActivity(ctx, "Received", "VNC Access Address", req.AppId, req.ClusterId, req.VirtualMachineId)
	return &resourceapiv2.GetVNCResponse{
		Address: vncAddress,
	}, nil

}

func (s *Server) RestartVirtualMachine(ctx context.Context, req *resourceapiv2.RestartVirtualMachineRequest) (*resourceapiv2.RestartVirtualMachineResponse, error) {
	log.Infow("Received restart virtual machine request", dazl.Stringer("request", req))
	if req == nil || req.AppId == "" || req.ClusterId == "" || req.VirtualMachineId == "" {
		errMsg := "Request is not valid; app ID or cluster ID or VM ID is empty"
		log.Warnw(errMsg, dazl.Stringer("request", req))
		return nil, errors.Status(errors.NewInvalid(errMsg)).Err()
	}
	err := req.ValidateAll()
	if err != nil {
		log.Warnw("Request validation is failed", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	// Validate ActiveProjectID is present and valid
	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Warnw("ActiveProjectID validation failed", dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if err := opa.IsAuthorized(ctx, req, s.opaClient); err != nil {
		log.Warnw("Access denied by OPA rules",
			dazl.String("AppID", req.AppId),
			dazl.String("ProjectID", activeProjectID),
			dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}

	err = s.sbHandler.RestartVM(ctx, req.AppId, req.ClusterId, req.VirtualMachineId)
	if err != nil {
		log.Warnw("Failed to restart virtual machine", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}
	logActivity(ctx, "Restarted", "VM", req.AppId, req.ClusterId, req.VirtualMachineId)
	return &resourceapiv2.RestartVirtualMachineResponse{}, nil
}

func (s *Server) StartVirtualMachine(ctx context.Context, req *resourceapiv2.StartVirtualMachineRequest) (*resourceapiv2.StartVirtualMachineResponse, error) {
	log.Infow("Received start virtual machine request", dazl.Stringer("request", req))
	if req == nil || req.AppId == "" || req.ClusterId == "" || req.VirtualMachineId == "" {
		errMsg := "Request is not valid; app ID or cluster ID or VM ID is empty"
		log.Warnw(errMsg, dazl.Stringer("request", req))
		return nil, errors.Status(errors.NewInvalid(errMsg)).Err()
	}

	err := req.ValidateAll()
	if err != nil {
		log.Warnw("Request validation is failed", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	// Validate ActiveProjectID is present and valid
	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Warnw("ActiveProjectID validation failed", dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if err := opa.IsAuthorized(ctx, req, s.opaClient); err != nil {
		log.Warnw("Access denied by OPA rules",
			dazl.String("AppID", req.AppId),
			dazl.String("ProjectID", activeProjectID),
			dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}

	err = s.sbHandler.StartVM(ctx, req.AppId, req.ClusterId, req.VirtualMachineId)
	if err != nil {
		log.Warnw("Failed to start virtual machine", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()

	}

	logActivity(ctx, "Started", "VM", req.AppId, req.ClusterId, req.VirtualMachineId)
	return &resourceapiv2.StartVirtualMachineResponse{}, nil
}

func (s *Server) StopVirtualMachine(ctx context.Context, req *resourceapiv2.StopVirtualMachineRequest) (*resourceapiv2.StopVirtualMachineResponse, error) {
	log.Infow("Received stop virtual machine request", dazl.Stringer("request", req))
	if req == nil || req.AppId == "" || req.ClusterId == "" || req.VirtualMachineId == "" {
		errMsg := "Request is not valid; app ID or cluster ID or VM ID is empty"
		log.Warnw(errMsg, dazl.Stringer("request", req))
		return nil, errors.Status(errors.NewInvalid(errMsg)).Err()
	}

	err := req.ValidateAll()
	if err != nil {
		log.Warnw("Request validation is failed", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	// Validate ActiveProjectID is present and valid
	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Warnw("ActiveProjectID validation failed", dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if err := opa.IsAuthorized(ctx, req, s.opaClient); err != nil {
		log.Warnw("Access denied by OPA rules",
			dazl.String("AppID", req.AppId),
			dazl.String("ProjectID", activeProjectID),
			dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}

	err = s.sbHandler.StopVM(ctx, req.AppId, req.ClusterId, req.VirtualMachineId)
	if err != nil {
		log.Warnw("Failed to stop virtual machine", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}
	logActivity(ctx, "Stopped", "VM", req.AppId, req.ClusterId, req.VirtualMachineId)
	return &resourceapiv2.StopVirtualMachineResponse{}, nil
}
