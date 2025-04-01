// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc/metadata"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	OPAEnabled = "OPA_ENABLED"
	// OIDCServerURL - address of an OpenID Connect server
	OIDCServerURL = "OIDC_SERVER_URL"
	OPAPort       = "OPA_PORT"
)

var log = dazl.GetPackageLogger()

// IsOPAEnabled checks if OPA deployment is enabled
func IsOPAEnabled() bool {
	enabled := os.Getenv(OPAEnabled)
	return enabled == "true"
}

func NewOPAClient(opaScheme, opaHostname string) openpolicyagent.ClientWithResponsesInterface {
	var opaClient openpolicyagent.ClientWithResponsesInterface
	oidcURL := os.Getenv(OIDCServerURL)
	if oidcURL == "" || !IsOPAEnabled() {
		log.Infow("Authentication and Authorization are not enabled", dazl.String(OIDCServerURL, oidcURL))
		return nil
	}

	log.Infow("Authentication and Authorization are enabled", dazl.String(OIDCServerURL, oidcURL))
	opaPortString := os.Getenv(OPAPort)
	opaPort, err := strconv.Atoi(opaPortString)
	if err != nil {
		log.Fatalw("OPA Port is no valid", dazl.Error(err))
		return nil
	}
	opaServerAddr := fmt.Sprintf("%s://%s:%d", opaScheme, opaHostname, opaPort)
	log.Infow("OPA is enabled, creating an OPA client", dazl.String("OPA server addr", opaServerAddr))

	opaClient, err = openpolicyagent.NewClientWithResponses(opaServerAddr)
	if err != nil {
		log.Fatalw("OPA client cannot be created", dazl.Error(err))
		return nil
	}

	return opaClient
}

func GetActiveProjectID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.NewInvalid("unable to extract metadata from ctx")
	}

	activeProjectIDs := md.Get("activeprojectid")

	if len(activeProjectIDs) > 1 {
		return "", errors.NewInvalid("multiple ActiveProjectIDs are set - it should be one")
	}

	// TODO: will be removed once fully MT is integrated, as activeProjectID is mandatory field for the future but not now
	if len(activeProjectIDs) == 0 {
		return "", nil
	}

	return activeProjectIDs[0], nil
}

func IsAuthorized(ctx context.Context, request interface{}, opaClient openpolicyagent.ClientWithResponsesInterface) error {
	if opaClient == nil {
		return nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.NewInvalid("unable to extract metadata from ctx")
	}
	opaInputStruct := openpolicyagent.OpaInput{
		Input: map[string]interface{}{
			"request":  request,
			"metadata": md,
		},
	}

	completeInputJSON, err := json.Marshal(opaInputStruct)
	if err != nil {
		return errors.NewInvalid("cannot marshal object to JSON %v %v", err, opaInputStruct)
	}

	bodyReader := bytes.NewReader(completeInputJSON)

	// The name of the protobuf request is an easy way of linking to REGO rules e.g. "*catalogv1.CreatePublisherRequest"
	requestType := reflect.TypeOf(request).String()
	requestPackage := requestType[1:strings.LastIndex(requestType, ".")]
	requestName := requestType[strings.LastIndex(requestType, ".")+1:]
	trueBool := true
	resp, err := opaClient.PostV1DataPackageRuleWithBodyWithResponse(
		ctx,
		requestPackage,
		requestName,
		&openpolicyagent.PostV1DataPackageRuleParams{
			Pretty:  &trueBool,
			Metrics: &trueBool,
		},
		"application/json",
		bodyReader)
	if err != nil {
		return errors.NewInvalid("error on OPA Post. %v", err)
	}

	resultBool, boolErr := resp.JSON200.Result.AsOpaResponseResult1()
	if boolErr != nil {
		resultObj, objErr := resp.JSON200.Result.AsOpaResponseResult0()
		if objErr != nil {
			errMsg := fmt.Sprintf("access denied by OPA rule %s %v %v", requestName, resultObj, objErr)
			log.Debugf(errMsg)
			return errors.NewInvalid(errMsg)
		}
		errMsg := fmt.Sprintf("access denied by OPA rule %s %v %v", requestName, resultObj, boolErr)
		log.Debugf(errMsg)
		return errors.NewForbidden(errMsg)

	}
	if resultBool {
		log.Infow("Authorized", dazl.String("requestName", requestName))
		return nil
	}

	log.Debugf("access denied by OPA rule %s. OPA response %d %v", requestName, resp.StatusCode(), resp.HTTPResponse)
	return errors.NewForbidden("access denied")
}
