// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/server"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	var (
		addr string
	)
	flag.StringVar(&addr, "listen", ":8123", "Listen address")
	flag.Parse()

	asp, err := server.NewServer(addr)
	if asp == nil {
		panic(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Create an error channel
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			// Handle the context being canceled
			fmt.Println("Context canceled")
		default:
			// Catch any error from Run and send it to the error channel
			if err := asp.Run(); err != nil {
				errChan <- err
			}
		}
	}()

	// Wait for either an error or an OS signal
	select {
	case err := <-errChan:
		// Handle the error
		fmt.Printf("Error encountered: %s\n", err)
	case sig := <-c:
		// Handle the signal
		fmt.Printf("Got %s signal. Aborting...\n", sig)
	}
}
