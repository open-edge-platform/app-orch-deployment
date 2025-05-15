// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html/atom"
	"k8s.io/apimachinery/pkg/util/sets"
)

// TestAtomsToAttrs verifies the mapping of HTML atoms to attributes that require URL substitution.
func TestAtomsToAttrs(t *testing.T) {
	expected := map[atom.Atom]sets.Set[string]{
		atom.A:          sets.New("href"),
		atom.Applet:     sets.New("codebase"),
		atom.Area:       sets.New("href"),
		atom.Audio:      sets.New("src"),
		atom.Base:       sets.New("href"),
		atom.Blockquote: sets.New("cite"),
		atom.Body:       sets.New("background"),
		atom.Button:     sets.New("formaction"),
		atom.Command:    sets.New("icon"),
		atom.Del:        sets.New("cite"),
		atom.Embed:      sets.New("src"),
		atom.Form:       sets.New("action"),
		atom.Frame:      sets.New("longdesc", "src"),
		atom.Head:       sets.New("profile"),
		atom.Html:       sets.New("manifest"),
		atom.Iframe:     sets.New("longdesc", "src"),
		atom.Img:        sets.New("longdesc", "src", "usemap"),
		atom.Input:      sets.New("src", "usemap", "formaction"),
		atom.Ins:        sets.New("cite"),
		atom.Link:       sets.New("href"),
		atom.Object:     sets.New("classid", "codebase", "data", "usemap"),
		atom.Q:          sets.New("cite"),
		atom.Script:     sets.New("src"),
		atom.Source:     sets.New("src"),
		atom.Video:      sets.New("poster", "src"),
	}
	for atom, attrs := range atomsToAttrs {
		expAttrs, ok := expected[atom]
		if !ok {
			t.Errorf("Unexpected atom in atomsToAttrs: %v", atom)
			continue
		}
		for attr := range expAttrs {
			if !attrs.Has(attr) {
				t.Errorf("Expected attribute %s for atom %v not found", attr, atom)
			}
		}
	}
}

// TestRewriteURL tests the rewriteURL function with various scenarios.
func TestRewriteURL(t *testing.T) {
	tests := []struct {
		description string
		url         string
		sourceURL   string
		cookieInfo  *CookieInfo
		kubeapiAddr *url.URL
		expected    string
	}{
		// Define test cases
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			url, _ := url.Parse(test.url)
			sourceURL, _ := url.Parse(test.sourceURL)
			result := rewriteURL(url, sourceURL, test.cookieInfo, test.kubeapiAddr)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestRewriteHTML verifies that HTML content is correctly rewritten.
func TestRewriteHTML(t *testing.T) {
	// Mock HTML content with links that need to be rewritten.
	sourceHTML := `<html>
	<head><title>Test</title></head>
	<body>
		<a href="http://kubernetes.default.svc/api/v1/namespaces/fake-namespace/services/fake-service:80/proxy/api/resource">Link</a>
		<img src="http://kubernetes.default.svc/api/v1/namespaces/fake-namespace/services/fake-service:80/proxy/api/image.png"/>
		<img src="/api/v1/namespaces/fake-namespace/services/fake-service:80/proxy/api/image.png"/>
	</body>
	</html>`
	expectedHTML := `<html>
	<head><title>Test</title></head>
	<body>
		<a href="http://app-service-proxy.kind.internal/api/resource">Link</a>
		<img src="http://app-service-proxy.kind.internal/api/image.png"/>
		<img src="//app-service-proxy.kind.internal/api/image.png"/>
	</body>
	</html>`

	// Setup reader from the source HTML and a writer to capture the output.
	reader := strings.NewReader(sourceHTML)
	var writer strings.Builder

	// Define cluster ID and source URL for URL rewriting.
	testCi := &CookieInfo{
		projectID: "fake-project",
		namespace: "fake-namespace",
		service:   "fake-service:80",
		cluster:   "cluster123",
	}
	target, _ := url.Parse("http://kubernetes.default.svc")
	sourceURL, _ := url.Parse("https://app-service-proxy.kind.internal")

	// Call rewriteHTML with our mock HTML content.
	err := rewriteHTML(reader, &writer, testCi, target, sourceURL)
	require.NoError(t, err)

	// Parse the output HTML and verify URLs.
	require.Equal(t, expectedHTML, writer.String(), "The HTML content was not rewritten as expected")
}

// TestRoundTrip tests the RoundTrip method of RewritingTransport.
func TestRoundTrip(t *testing.T) {
	// Create a new router
	router := mux.NewRouter()

	// Define a subrouter for handling paths with the cluster prefix
	proxyRouter := router.PathPrefix("/projects").Subrouter()
	// Define the handler for the path pattern including `clusterId`
	proxyRouter.HandleFunc("{project}/clusters/{cluster}/api/resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body><a href="/api/resource">Link</a></body></html>`)
	}).Methods("GET")

	// Create a test server using the router
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	// Use the test server URL to construct the request URL, including the clusterId segment
	requestURL := testServer.URL + "/projects/project1/clusters/cluster123/api/resource"

	// Create the HTTP request to the test server
	req, err := http.NewRequest("GET", requestURL, nil)
	require.NoError(t, err)

	// Add X-Forwarded-Proto and X-Forwarded-Host headers
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "original.example.com")

	// Create an instance of RewritingTransport with the test server as the transport
	transport := &RewritingTransport{}

	// Perform the round trip using the RewritingTransport
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Assert that the response body has been rewritten as expected
	expectedBody := `<html><body><a href="/projects/project1/clusters/cluster123/api/resource">Link</a></body></html>`
	assert.NotEqual(t, expectedBody, string(body), "The response body was not rewritten as expected")

}

// TestRewriteResponse checks the rewriteResponse function's ability to modify responses correctly.
func TestRewriteResponse(t *testing.T) {
	// Setup a RewritingTransport, mock request and response
	// Verify that the response body is modified as expected
	// Mock an original response that includes URLs to be rewritten
	originalHTML := `<html><body><a href="/api/v1/namespaces/namespace123/services/service123:80/proxy/api/resource">Link</a></body></html>`
	expectedHTML := `<html><body><a href="//original.example.com/api/resource">Link</a></body></html>`
	var originalBody io.Reader = strings.NewReader(originalHTML)
	// Simulate a gzip compression if necessary
	buf := new(bytes.Buffer)
	gz := gzip.NewWriter(buf)
	_, err := io.Copy(gz, originalBody)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	// Mock an HTTP response
	resp := &http.Response{
		Header:     make(http.Header),
		Body:       io.NopCloser(buf),
		StatusCode: http.StatusOK,
	}
	resp.Header.Set("Content-Encoding", "gzip")
	resp.Header.Set("Content-Type", "text/html")

	// Mock a request, including the mux variables to simulate extracting `clusterId`
	req := httptest.NewRequest("GET", "http://kubernetes.default.svc/api/resource", nil)

	// Add X-Forwarded-Proto and X-Forwarded-Host headers
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "original.example.com")
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})

	rt := RewritingTransport{}

	// Call rewriteResponse or the method that uses it internally
	modifiedResp, err := rt.rewriteResponse(req, resp)
	require.NoError(t, err)

	// Read the modified response body
	modifiedBody, err := io.ReadAll(modifiedResp.Body)
	require.NoError(t, err)

	// Decompress the gzip content if necessary
	modifiedReader := bytes.NewReader(modifiedBody)
	gzReader, err := gzip.NewReader(modifiedReader)
	require.NoError(t, err)
	decompressedBody, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	// Convert the body to a string for assertion
	modifiedHTML := string(decompressedBody)

	// Assert that the URLs within the response have been rewritten correctly
	assert.Equal(t, expectedHTML, modifiedHTML, "The response body was not rewritten as expected")
}

// TestRewriteHTMLComplex verifies rewriteHTML with complex HTML structures.
func TestRewriteHTMLComplex(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	sourceHTML := `<html>
<head><link rel="stylesheet" href="/api/v1/namespaces/namespace123/services/service123:80/proxy/api/style.css"></head>
<body>
	<style id="wp-fonts-local">
		@font-face{font-family:Inter;src:url('https://kubernetes.default.svc/api/v1/namespaces/namespace123/services/service123:80/proxy/wp-content/themes/twentytwentyfour/assets/fonts/inter/Inter-VariableFont_slnt,wght.woff2') format('woff2');font-stretch:normal;}
	</style>
</body>
</html>`
	expectedHTML := `<html>
<head><link rel="stylesheet" href="//app-service-proxy.kind.internal/api/style.css"></head>
<body>
	<style id="wp-fonts-local">
		@font-face{font-family:Inter;src:url('https://app-service-proxy.kind.internal/wp-content/themes/twentytwentyfour/assets/fonts/inter/Inter-VariableFont_slnt,wght.woff2') format('woff2');font-stretch:normal;}
	</style>
</body>
</html>`
	testCookie := &CookieInfo{
		projectID: "fake-project",
		cluster:   "cluster123",
		namespace: "namespace123",
		service:   "service123:80",
	}
	target, _ := url.Parse("https://kubernetes.default.svc")
	sourceURL, _ := url.Parse("https://app-service-proxy.kind.internal")

	reader := strings.NewReader(sourceHTML)
	var writer strings.Builder
	err := rewriteHTML(reader, &writer, testCookie, target, sourceURL)
	require.NoError(t, err)

	outputHTML := writer.String()

	require.Equal(t, expectedHTML, outputHTML, "The HTML content was not rewritten as expected")
	t.Log(buf.String())
}

// TestRoundTripContentTypes tests the RoundTrip method with various content types.
func TestRoundTripContentTypes(t *testing.T) {
	// Define test cases with different content types and expected behaviors
	tests := []struct {
		contentType     string
		content         string
		shouldRewrite   bool // Indicates whether the content is expected to be rewritten
		expectedContent string
	}{
		{
			contentType:     "text/html",
			content:         `<a href="/api/v1/namespaces/namespace123/services/service123:80/proxy/api/resource">Link</a>`,
			shouldRewrite:   true,
			expectedContent: `<a href="//original.example.com/api/resource">Link</a>`,
		},
		{
			contentType:     "application/json",
			content:         `{"link": "/api/resource"}`,
			shouldRewrite:   false,
			expectedContent: `{"link": "/api/resource"}`,
		},
		{
			contentType:     "text/plain",
			content:         "Visit /api/resource for more information.",
			shouldRewrite:   false,
			expectedContent: "Visit /api/resource for more information.",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("ContentType=%s", tc.contentType), func(t *testing.T) {
			// Create a test server that returns the specified content type and body
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				_, _ = io.WriteString(w, tc.content)
			}))
			defer testServer.Close()

			// Create an HTTP request to the test server
			req, err := http.NewRequest("GET", testServer.URL, nil)
			require.NoError(t, err)

			// Add X-Forwarded-Proto and X-Forwarded-Host headers
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Forwarded-Host", "original.example.com")

			// Add cookies to the request to simulate the cookie info
			req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
			req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
			req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
			req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})

			// Create an instance of RewritingTransport with the test server's client as the transport
			transport := &RewritingTransport{}

			// Perform the round trip using the RewritingTransport
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Assert that the response body is as expected, based on whether it should be rewritten
			if tc.shouldRewrite {
				assert.Equal(t, tc.expectedContent, string(body), "The response body for content type %s was not rewritten as expected", tc.contentType)
			} else {
				assert.Equal(t, tc.content, string(body), "The response body for content type %s should not have been modified", tc.contentType)
			}
		})
	}
}

// mockRoundTripper is a mock implementation of http.RoundTripper that always returns an error.
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("mock transport error")
}

// TestRoundTripErrorHandling tests error handling in RoundTrip.
func TestRoundTripErrorHandling(t *testing.T) {
	// Create a RewritingTransport instance with the mockRoundTripper.
	transport := &RewritingTransport{
		transport: &mockRoundTripper{},
	}

	// Create a dummy request.
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	// Add X-Forwarded-Proto and X-Forwarded-Host headers
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "original.example.com")

	// Perform the round trip.
	_, err = transport.RoundTrip(req)

	// Assert that an error was returned and check if it's the expected error.
	assert.Error(t, err, "Expected an error to be returned from RoundTrip")
	assert.Contains(t, err.Error(), "mock transport error", "Unexpected error message returned from RoundTrip")
}

// deflateContent compresses given content using flate (deflate) compression.
func deflateContent(content []byte) (*bytes.Buffer, error) {
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(content)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// TestRewritingTransportWithDeflate tests the RewritingTransport's handling of deflate-compressed content.
func TestRewritingTransportWithDeflate(t *testing.T) {
	// Original HTML content and expected rewritten content
	originalHTML := `<html><body><a href="/api/resource">Link</a></body></html>`
	expectedHTML := `<html><body><a href="/projects/project1/clusters/cluster123/api/resource">Link</a></body></html>`

	// Compress the original HTML content using deflate
	deflatedContent, err := deflateContent([]byte(originalHTML))
	require.NoError(t, err)

	// Create a test server that returns the deflated content with the appropriate Content-Encoding header
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "deflate")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(deflatedContent.Bytes())
	}))
	defer testServer.Close()

	// Create the HTTP request
	req, err := http.NewRequest("GET", testServer.URL, nil)
	require.NoError(t, err)

	// Add X-Forwarded-Proto and X-Forwarded-Host headers
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "original.example.com")

	// Add cookies to the request to simulate the cookie info
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-project", Value: "project1"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-cluster", Value: "cluster123"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-namespace", Value: "namespace123"})
	req.AddCookie(&http.Cookie{Name: "app-service-proxy-service", Value: "service123:80"})

	// Set up the RewritingTransport with the default transport
	transport := &RewritingTransport{
		transport: http.DefaultTransport,
	}

	// Perform the round trip
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Read and verify the rewritten HTML content
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEqual(t, expectedHTML, string(body), "The response body was not correctly rewritten")
}

// TestCSPFrameAncestorsPatched verifies that the CSP header is patched correctly.
func TestCSPFrameAncestorsPatched(t *testing.T) {
	originalCSP := "default-src 'self'; frame-ancestors 'none'; script-src 'self'"
	expectedCSP := "default-src 'self'; frame-ancestors 'self'; script-src 'self'"

	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("<html></html>")),
	}
	resp.Header.Set("Content-Security-Policy", originalCSP)
	resp.Header.Set("Content-Type", "text/html")

	// Use a dummy transport that returns our mocked response
	dummyTransport := &struct{ http.RoundTripper }{}
	rt := &RewritingTransport{transport: dummyTransport}
	rt.transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return resp, nil
	})

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Set("X-Forwarded-Proto", "http")

	gotResp, err := rt.RoundTrip(req)
	require.NoError(t, err)

	gotCSP := gotResp.Header.Get("Content-Security-Policy")
	assert.Equal(t, expectedCSP, gotCSP, "CSP header was not patched correctly")
}

// TestCSPFrameAncestorsAddsSelfIfMissing verifies that 'self' is added to frame-ancestors if missing.
func TestCSPFrameAncestorsAddsSelfIfMissing(t *testing.T) {
	originalCSP := "default-src 'self'; frame-ancestors https://example.com; script-src 'self'"
	expectedCSP := "default-src 'self'; frame-ancestors https://example.com 'self'; script-src 'self'"

	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("<html></html>")),
	}
	resp.Header.Set("Content-Security-Policy", originalCSP)
	resp.Header.Set("Content-Type", "text/html")

	dummyTransport := &struct{ http.RoundTripper }{}
	rt := &RewritingTransport{transport: dummyTransport}
	rt.transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return resp, nil
	})

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Set("X-Forwarded-Proto", "http")

	gotResp, err := rt.RoundTrip(req)
	require.NoError(t, err)

	gotCSP := gotResp.Header.Get("Content-Security-Policy")
	assert.Equal(t, expectedCSP, gotCSP, "CSP header was not patched correctly when 'self' was missing")
}

// Helper type for inline RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
