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

func (s *Server) ListAppEndpoints(ctx context.Context, req *resourceapiv2.ListAppEndpointsRequest) (*resourceapiv2.ListAppEndpointsResponse, error) {
	log.Infow("Received List App Endpoints request", dazl.Stringer("request", req))
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

	appEndpoints, err := s.sbHandler.GetAppEndpointsV2(ctx, req.AppId, req.ClusterId)
	if err != nil {
		log.Warnw("Failed to list application endpoints", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}

	logActivity(ctx, "Received", "App Endpoints List", req.AppId, req.ClusterId)
	resp := &resourceapiv2.ListAppEndpointsResponse{
		AppEndpoints: appEndpoints,
	}

	return resp, nil

}
