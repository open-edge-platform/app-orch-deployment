// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package adm

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"google.golang.org/grpc/metadata"
	"time"
)

func getCtxWithToken(ctx context.Context, vaultAuthClient auth.VaultAuth) (context.Context, error) {
	token, err := vaultAuthClient.GetM2MToken(ctx)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	outCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	err = vaultAuthClient.Logout(outCtx)
	return outCtx, err
}

func addToOutgoingContext(ctx context.Context, vaultAuthClient auth.VaultAuth, setTimeout bool) (context.Context, context.CancelFunc, error) {
	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Warn(err)
		return ctx, nil, err
	}

	ctx, err = getCtxWithToken(ctx, vaultAuthClient)
	if err != nil {
		log.Warn(err)
		return ctx, nil, err
	}

	outCtx := metadata.AppendToOutgoingContext(ctx, "ActiveProjectID", activeProjectID)
	if setTimeout {
		ctx, cancel := context.WithTimeout(outCtx, 30*time.Second)
		return ctx, cancel, nil
	}

	return outCtx, nil, nil
}
