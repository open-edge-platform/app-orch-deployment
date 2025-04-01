// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleetclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	runtime "k8s.io/apimachinery/pkg/runtime"
)

func NewRequestHandler(retObj runtime.Object, statusCode int) func(req *http.Request) (*http.Response, error) {
	return func(_ *http.Request) (*http.Response, error) {
		retData, err := json.Marshal(retObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshall desired return Object: %w", err)
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewReader(retData)),
		}, nil
	}
}
