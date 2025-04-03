// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	authlib "github.com/open-edge-platform/orch-library/go/pkg/grpc/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	opaHostname = "localhost"
	OPAPort     = "OPA_PORT"
	opaScheme   = "http"
)

const (
	tokenCookiePrefix = "app-service-proxy-token" // #nosec G101
	tokenCookieCount  = tokenCookiePrefix + "s"
)

var AuthenticateFunc = Authenticate
var AuthorizeFunc = Authorize

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func Authenticate(req *http.Request) error {
	authToken, err := getCombinedJWTToken(req)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot get JWT token to Authenticate %v", err)
	}
	jwtAuth := new(auth.JwtAuthenticator)
	_, err = jwtAuth.ParseAndValidate(authToken)
	if err != nil {
		logrus.Warnf("Failed to validate JWT token %v", err)
		return err
	}
	return nil
}

// getCombinedJWTToken extracts all cookies with the name "token" and combines their values to form a JWT token
// It also removes the Cookie so that it is not sent to the proxied application
func getCombinedJWTToken(req *http.Request) (string, error) {
	var tokenParts []string

	numTokenCookiesStr, err := req.Cookie(tokenCookieCount)
	if err != nil {
		logrus.Errorf("Error retrieving %s cookie: %v", tokenCookieCount, err)
		return "", err
	}
	// Delete the cookie by replacing it with an expired one
	req.AddCookie(&http.Cookie{Name: tokenCookieCount, Expires: time.Unix(0, 0), MaxAge: -1})

	// Iterate over all cookies in the request
	numTokenCookies, err := strconv.Atoi(numTokenCookiesStr.Value)
	if err != nil {
		logrus.Errorf("Error parsing %s cookie value: %v", tokenCookieCount, err)
		return "", err
	}
	for num := 0; num < numTokenCookies; num++ {
		tokenPartName := tokenCookiePrefix + "-" + strconv.Itoa(num)
		logrus.Infof("found tokenName : %s", tokenPartName)
		tokenPartValue, err := req.Cookie(tokenPartName)
		if err != nil {
			logrus.Errorf("Error retrieving %s cookie: %v", tokenPartName, err)
			return "", err
		}
		// Delete the cookie by replacing it with an expired one
		req.AddCookie(&http.Cookie{Name: tokenPartName, Expires: time.Unix(0, 0), MaxAge: -1})

		tokenParts = append(tokenParts, tokenPartValue.Value)
	}

	// Combine the token parts to form the JWT token
	jwtToken := strings.Join(tokenParts, "")
	return jwtToken, nil
}

func Authorize(req *http.Request, projectID string) error {
	opaClient := getOpaClient()
	if opaClient == nil {
		return status.Errorf(codes.Internal, "cannot get opaclient")
	}

	ctx := context.Background()

	jwt := ""
	var err error

	jwt, err = getCombinedJWTToken(req)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot get JWT token to Authorize %v", err)
	}

	md := metadata.New(map[string]string{"authorization": fmt.Sprintf("Bearer %s", jwt), "activeprojectid": projectID})
	ctx = metadata.NewIncomingContext(ctx, md)
	ctx, err = authlib.AuthenticationInterceptor(ctx)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse token string, error: %v", err)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "unable to extract metadata from ctx")
	}

	opaInputStruct := openpolicyagent.OpaInput{
		Input: map[string]interface{}{
			"method":   req.Method,
			"metadata": md,
		},
	}

	completeInputJSON, err := json.Marshal(opaInputStruct)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot marshal object to JSON %v %v", err, opaInputStruct)
	}
	bodyReader := bytes.NewReader(completeInputJSON)
	trueBool := true
	resp, err := opaClient.PostV1DataPackageRuleWithBodyWithResponse(
		ctx,
		"deploymentv1",
		"allow",
		&openpolicyagent.PostV1DataPackageRuleParams{
			Pretty:  &trueBool,
			Metrics: &trueBool,
		},
		"application/json",
		bodyReader)

	if err != nil {
		return status.Errorf(codes.InvalidArgument, "error on OPA Post. %v %v", err, resp.JSONDefault)
	}

	resultBool, boolErr := resp.JSON200.Result.AsOpaResponseResult1()
	if boolErr != nil {
		resultObj, objErr := resp.JSON200.Result.AsOpaResponseResult0()
		if objErr != nil {
			logrus.Warnf("access denied by OPA rule %v", objErr)
			return status.Errorf(codes.PermissionDenied, "access denied")
		}

		logrus.Warnf("access denied by OPA rule %v", resultObj)
		return status.Errorf(codes.PermissionDenied, "access denied")

	}
	if resultBool {
		return nil
	}

	logrus.Warnf("access denied by OPA: %d %v", resp.StatusCode(), resp.HTTPResponse)
	return status.Errorf(codes.PermissionDenied, "access denied")
}

func IsAuthZenabled() bool {
	if os.Getenv("OPA_ENABLED") != "true" {
		return false
	}
	opaPortString := os.Getenv(OPAPort)
	_, err := strconv.Atoi(opaPortString)
	if err != nil {
		logrus.Fatalf("OPA Port is not valid %v", err)
		return false
	}
	return true
}

func getOpaClient() *openpolicyagent.ClientWithResponses {
	opaPortString := os.Getenv(OPAPort)
	opaPort, err := strconv.Atoi(opaPortString)
	if err != nil {
		logrus.Fatalf("OPA Port is not valid %v", err)
		return nil
	}
	serverAddr := fmt.Sprintf("%s://%s:%d", opaScheme, opaHostname, opaPort)
	opaClient, err := openpolicyagent.NewClientWithResponses(serverAddr)
	if err != nil {
		logrus.Fatalf("OPA client cannot be created %v", err)
		return nil
	}
	return opaClient
}
