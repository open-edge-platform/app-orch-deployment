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

func (s *Server) ListAppWorkloads(ctx context.Context, req *resourceapiv2.ListAppWorkloadsRequest) (*resourceapiv2.ListAppWorkloadsResponse, error) {
	log.Infow("Received List App Workload request", dazl.Stringer("request", req))
	if req == nil || req.AppId == "" || req.ClusterId == "" {
		errMsg := "Request is not valid; app ID or cluster ID is empty"
		log.Warnw(errMsg, dazl.Stringer("request", req))
		return nil, errors.Status(errors.NewInvalid(errMsg)).Err()
	}
	err := req.ValidateAll()
	if err != nil {
		log.Warnw("Request validation is failed", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if err := opa.IsAuthorized(ctx, req, s.opaClient); err != nil {
		log.Warnw("Access denied by OPA rules", dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}
	appWorkloads, err := s.sbHandler.GetAppWorkLoads(ctx, req.AppId, req.ClusterId)
	if err != nil {
		log.Warnw("Failed to list application workloads", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}
	logActivity(ctx, "Received", "App Workload List", req.AppId, req.ClusterId)
	resp := &resourceapiv2.ListAppWorkloadsResponse{
		AppWorkloads: appWorkloads,
	}

	return resp, nil

}
