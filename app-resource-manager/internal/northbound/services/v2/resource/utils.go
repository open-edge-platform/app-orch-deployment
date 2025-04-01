// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"google.golang.org/grpc/metadata"
	"strings"
)

func logActivity(ctx context.Context, verb string, thing string, args ...string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok && len(md.Get("name")) > 0 {
		log.Infow("User", dazl.Strings("name", md.Get("name")),
			dazl.String("verb", verb),
			dazl.String("thing", thing),
			dazl.String("args", strings.Join(args, "/")))
	} else {
		log.Infow("User", dazl.Strings("client", md.Get("client")),
			dazl.String("verb", verb),
			dazl.String("thing", thing),
			dazl.String("args", strings.Join(args, "/")))
	}
}
