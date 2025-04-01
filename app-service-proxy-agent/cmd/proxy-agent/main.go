// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/agent"
)

func main() {
	ctx := context.Background()

	agent, err := agent.NewServiceProxyAgent()
	if err != nil {
		fmt.Printf("failed to start the service-proxy agent (%v)", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	agent.Run(ctx)

	sig := <-c
	fmt.Printf("Got %s signal. Aborting...\n", sig)
	agent.Stop()
}
