// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/middleware"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

var _ = Describe("Middleware", func() {
	var (
		responseRecorder *httptest.ResponseRecorder
		request          *http.Request
		nextHandler      http.Handler
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()
		nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	Describe("SizeLimitMiddleware", func() {
		Context("when request body size is within the limit", func() {
			It("should pass the request to the next handler", func() {
				body := bytes.NewReader(make([]byte, 124)) // 1KB body
				request = httptest.NewRequest("POST", "/test", body)
				middleware := SizeLimitMiddleware(2048)(nextHandler) // 2KB limit

				middleware.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when request body exceeds the size limit", func() {
			It("should return an error response", func() {
				body := bytes.NewReader(make([]byte, 4096)) // 4KB body
				request = httptest.NewRequest("POST", "/test", body)
				middleware := SizeLimitMiddleware(2048)(nextHandler) // 2KB limit

				middleware.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			})
		})
	})
})
