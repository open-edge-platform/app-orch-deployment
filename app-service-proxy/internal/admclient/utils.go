// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package admclient

import (
	"context"
	"github.com/sirupsen/logrus"

	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"google.golang.org/grpc/metadata"
	"time"
)

func getCtxWithToken(ctx context.Context, vaultAuthClient auth.VaultAuth) (context.Context, context.CancelFunc, error) {
	token, err := vaultAuthClient.GetM2MToken(ctx)
	if err != nil {
		return nil, nil, err
	}

	if token == "" {
		logrus.Error("token is empty")
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err = vaultAuthClient.Logout(ctx)
	return ctx, cancel, err
}
