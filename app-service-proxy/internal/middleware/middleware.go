// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
)

// SizeLimitMiddleware returns a middleware function that limits request body size
// The limit parameter specifies the maximum allowed size in bytes.
func SizeLimitMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit the size of the request body to the specified limit
			r.Body = http.MaxBytesReader(w, r.Body, limit)

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	}
}
