// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

// TestOnlyFunction is a function that is only available during testing.
func TestSetAuthorization(testServer *Server) {
	fmt.Printf("stub authorization\n")
	testServer.authNenabled = false
	testServer.authZenabled = false
	testServer.Authorize = func(req *http.Request, projectID string) error {
		fmt.Printf("stub authorization inside\n")
		return nil
	}
}

func TestSetAuthentication(testServer *Server) {
	fmt.Printf("stub authentication\n")
	testServer.Authenticate = func(req *http.Request) error {
		fmt.Printf("stub authentication inside\n")
		return nil
	}
}

// TestOnlyFunction is a function that is only available during testing.
func TestFailAuthorization(testServer *Server) {
	fmt.Printf("fail authorization\n")
	testServer.Authorize = func(req *http.Request, projectID string) error {
		return status.Errorf(codes.InvalidArgument, "Authorize fail")
	}
}

func TestFailAuthentication(testServer *Server) {
	fmt.Printf("fail authentication\n")
	testServer.Authenticate = func(req *http.Request) error { return status.Errorf(codes.InvalidArgument, "Authenticate fail") }
}
