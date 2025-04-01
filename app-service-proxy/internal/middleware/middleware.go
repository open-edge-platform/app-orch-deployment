// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"unicode"
)

func ValidateUnicodePrintableChars(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the entire request body into a buffer
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			http.Error(w, "copy error", http.StatusInternalServerError)
			return
		}

		// Create two ReadCloser instances from the buffer
		rdr1 := ioutilNopCloser(buf.Bytes())
		rdr2 := ioutilNopCloser(buf.Bytes())

		// Create a bufio.Reader to read runes from rdr1
		reader := bufio.NewReader(rdr1)

		// Perform validation on the request body using rdr1
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				if err == io.EOF {
					break
				}
				http.Error(w, "read error", http.StatusInternalServerError)
				return
			}
			if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
		}

		// Assign rdr2 back to Request.Body to preserve the original request body
		r.Body = rdr2

		// Continue with the next handler if the request is valid
		next.ServeHTTP(w, r)
	})
}

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

// ioutilNopCloser creates a ReadCloser with a no-op Close method
func ioutilNopCloser(b []byte) io.ReadCloser {
	return &nopCloser{bytes.NewReader(b)}
}

type nopCloser struct {
	io.Reader
}

func (n *nopCloser) Close() error {
	return nil
}
