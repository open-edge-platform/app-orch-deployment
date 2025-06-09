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

func (s *Server) DeletePod(ctx context.Context, req *resourceapiv2.DeletePodRequest) (*resourceapiv2.DeletePodResponse, error) {
	log.Infow("Received Delete Pod request", dazl.Stringer("request", req))
	if req == nil || req.PodName == "" || req.Namespace == "" || req.ClusterId == "" {
		errMsg := "Request is not valid; pod name or namespace or cluster ID is empty"
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
			dazl.String("PodName", req.PodName),
			dazl.String("Namespace", req.Namespace),
			dazl.String("ClusterID", req.ClusterId),
			dazl.String("ProjectID", activeProjectID),
			dazl.Error(err))
		return nil, errors.Status(errors.NewForbidden(err.Error())).Err()
	}

	err = s.sbHandler.DeletePod(ctx, req.ClusterId, req.Namespace, req.PodName)
	if err != nil {
		log.Warnw("Failed to delete pod", dazl.Stringer("request", req), dazl.Error(err))
		return nil, errors.Status(err).Err()
	}
	logActivity(ctx, "Received", "Delete Pod", req.PodName)
	resp := &resourceapiv2.DeletePodResponse{}

	return resp, nil

}
