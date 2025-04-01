// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/token"
	"os"
	"os/signal"
)

func main() {
	ctx := context.Background()
	tokenManager := token.NewManager()
	tokenManager.Start(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	sig := <-c
	fmt.Printf("Got %s signal. Aborting...\n", sig)
}
