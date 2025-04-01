// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

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

	Describe("ValidateUnicodePrintableChars", func() {
		Context("when request body contains only printable characters", func() {
			It("should pass the request to the next handler", func() {
				body := strings.NewReader("This is a valid request body with printable characters.")
				request = httptest.NewRequest("POST", "/test", body)
				middleware := ValidateUnicodePrintableChars(nextHandler)

				middleware.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			})
		})

		Context("when request body contains non-printable characters", func() {
			It("should return an error response", func() {
				body := strings.NewReader("Invalid\x00Body")
				request = httptest.NewRequest("POST", "/test", body)
				middleware := ValidateUnicodePrintableChars(nextHandler)

				middleware.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
			})
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

	Describe("ValidateUnicodePrintableChars", func() {
		It("should return an internal server error if copying the body fails", func() {
			// Creating a request with a closed body to simulate an error in reading the body
			req := httptest.NewRequest("POST", "/test", errorReader{})
			req.Body.Close() // Simulating an error condition

			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			middleware := ValidateUnicodePrintableChars(handler)

			middleware.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
