// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"bytes"
	"context"
	"encoding/json"

	"reflect"
	"strings"

	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc/metadata"
)

func (s *DeploymentSvc) GetActiveProjectID(ctx context.Context) (string, error) {
	if s.opaClient == nil {
		return "", nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.NewInvalid("unable to extract metadata from ctx")
	}
	activeProjectIDs := md.Get("activeprojectid")

	if len(activeProjectIDs) > 1 {
		return "", errors.NewInvalid("multiple ActiveProjectIDs are set - it should be one")
	}

	if len(activeProjectIDs) == 0 {
		return "", errors.NewInvalid("activeprojectid is not set")
	}

	return activeProjectIDs[0], nil
}

func (s *DeploymentSvc) AuthCheckAllowed(ctx context.Context, request any) error {
	if s.opaClient == nil {
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.NewInvalid("unable to extract metadata from ctx")
	}
	opaInputStruct := openpolicyagent.OpaInput{
		Input: map[string]any{
			"request":  request,
			"metadata": md,
		},
	}

	completeInputJSON, err := json.Marshal(opaInputStruct)
	if err != nil {
		return errors.NewInvalid("cannot marshal object to JSON %v %v", err, opaInputStruct)
	}

	bodyReader := bytes.NewReader(completeInputJSON)

	// The name of the protobuf request is an easy way of linking to REGO rules e.g. "*deploymentv1.CreateDeploymentRequest"
	requestType := reflect.TypeOf(request).String()
	requestPackage := requestType[1:strings.LastIndex(requestType, ".")]
	requestName := requestType[strings.LastIndex(requestType, ".")+1:]

	trueBool := true
	resp, err := s.opaClient.PostV1DataPackageRuleWithBodyWithResponse(
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
		return errors.NewInvalid("OPA rule %s OPA Post error %v", requestName, err)
	}

	resultBool, boolErr := resp.JSON200.Result.AsOpaResponseResult1()
	if boolErr != nil {
		resultObj, objErr := resp.JSON200.Result.AsOpaResponseResult0()
		if objErr != nil {
			log.Debugf("access denied (1) by OPA rule %s %v", requestName, objErr)
			return errors.NewForbidden("access denied (1)")
		}

		log.Debugf("access denied (2) by OPA rule %s %v", requestName, resultObj)
		return errors.NewForbidden("access denied (2)")

	}
	if resultBool {
		log.Debugf("%s Authorized", requestName)
		return nil
	}

	log.Debugf("access denied (3) by OPA rule %s. OPA response %d %v", requestName, resp.StatusCode(), resp.HTTPResponse)
	return errors.NewForbidden("access denied (3)")
}
