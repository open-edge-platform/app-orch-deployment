// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"k8s.io/apimachinery/pkg/util/sets"
)

// atomsToAttrs states which attributes of which tags require URL substitution.
// Sources: http://www.w3.org/TR/REC-html40/index/attributes.html
//
//	http://www.w3.org/html/wg/drafts/html/master/index.html#attributes-1
var atomsToAttrs = map[atom.Atom]sets.Set[string]{
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

	// TODO: css URLs hidden in style elements.
}

// RewritingTransport wraps the default http.RoundTripper
type RewritingTransport struct {
	transport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction and allows for response manipulation
func (t *RewritingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Send the request using the default transport
	if t.transport == nil {
		t.transport = http.DefaultTransport
	}
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		logrus.Errorf("transport error: %s", req.URL)
		return nil, err
	}

	cType := resp.Header.Get("Content-Type")
	cType = strings.TrimSpace(strings.SplitN(cType, ";", 2)[0])
	if cType != "text/html" {
		// Do nothing, simply pass through
		logrus.Debugf("Ignoring content of type: %s", resp.Header.Get("Content-Type"))
		return resp, nil
	}

	return t.rewriteResponse(req, resp)
}

func rewriteURL(url *url.URL, sourceURL *url.URL, ci *CookieInfo, kubeapiAddr *url.URL) string {
	isDifferentHost := url.Host != "" && url.Host != kubeapiAddr.Host
	isRelative := url.Path != "" && !strings.HasPrefix(url.Path, "/")
	if isDifferentHost || isRelative {
		logrus.Debugf("non updated url : %s:%s", url.Host, url.Path)
		return url.String()
	}

	// Example: /api/v1/namespaces/deployment-lnfs7/services/b-0cde7bf8-7ae1-5d13-835a-4bf3238eff74-wordpress:80/proxy/sample-page/
	// Becomes: /sample-page/
	url.Path = strings.TrimPrefix(url.Path, fmt.Sprintf(`/api/v1/namespaces/%s/services/%s/proxy`, ci.namespace, ci.service))
	url.Host = sourceURL.Host

	return url.String()
}

// ModifyURL modifies the URLs according to the specified rules
func ModifyURL(u string, ci *CookieInfo, srcURL string, kubeapiAddr string) string {
	if strings.HasPrefix(u, kubeapiAddr) {
		ur, err := url.Parse(u)
		if err != nil {
			return u
		}
		ur.Path = strings.TrimPrefix(ur.Path, fmt.Sprintf(`/api/v1/namespaces/%s/services/%s/proxy`, ci.namespace, ci.service))
		ur.Host = srcURL
		return ur.String()
	}
	return u
}

// ProcessJavaScript processes JavaScript content and modifies all URLs as needed
func processJavaScript(jsContent string, ci *CookieInfo, srcURL, kubeapiAddr string) string {
	// Grafana : Rewrite appSubUrl in JavaScripts

	appSubURLPattern := `"appSubUrl"\s*:\s*"(.*?)"`
	appSubURLRe := regexp.MustCompile(appSubURLPattern)
	modifiedJS := appSubURLRe.ReplaceAllStringFunc(jsContent, func(m string) string {
		// Extract the appSubUrl from the matched pattern
		url := appSubURLRe.FindStringSubmatch(m)[1]
		return ModifyURL(url, ci, srcURL, kubeapiAddr)
	})

	ur, err := url.Parse(kubeapiAddr)
	if err != nil {
		logrus.Debugf("parse kube addr failed")
		return modifiedJS
	}
	modifiedJSCode := strings.ReplaceAll(modifiedJS, ur.Host, srcURL)
	return modifiedJSCode
}

// ProcessCSS processes CSS content and modifies all URLs as needed
func processCSS(cssContent string, ci *CookieInfo, srcURL, kubeapiAddr string) string {
	// Updated pattern to handle newlines and whitespace characters
	pattern := fmt.Sprintf(`url\(\s*['"]?(%s/[^'")\s]+)\s*['"]?\)`, kubeapiAddr)
	re := regexp.MustCompile(pattern)
	modifiedCSS := re.ReplaceAllStringFunc(cssContent, func(m string) string {
		matches := re.FindStringSubmatch(m)
		if len(matches) > 1 {
			return "url('" + ModifyURL(matches[1], ci, srcURL, kubeapiAddr) + "')"
		}

		return m
	})

	return modifiedCSS
}

// rewriteHTML scans the HTML for tags with url-valued attributes, and updates
// those values with the rewriteURL function. The updated HTML is output to the
// writer.
func rewriteHTML(reader io.Reader, writer io.Writer, ci *CookieInfo, kubeapiAddr *url.URL, sourceURL *url.URL) error {
	// Note: This assumes the content is UTF-8.
	tokenizer := html.NewTokenizer(reader)
	var err error
	inStyleTag := false
	inScriptTag := false
	var styleContent, scriptContent bytes.Buffer

	for err == nil {
		tokenType := tokenizer.Next()
		// Print the token type in human readable form
		switch tokenType {
		case html.ErrorToken:
			err = tokenizer.Err()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if urlAttrs, ok := atomsToAttrs[token.DataAtom]; ok {
				for i, attr := range token.Attr {
					if urlAttrs.Has(attr.Key) {
						url, err := url.Parse(attr.Val)
						if err != nil {
							// Do not rewrite the URL if it isn't valid.  It is intended not
							// to error here to prevent the inability to understand the
							// content of the body to cause a fatal error.
							continue
						}
						token.Attr[i].Val = rewriteURL(url, sourceURL, ci, kubeapiAddr)
					}
				}
			}
			if token.DataAtom == atom.Style {
				inStyleTag = true
				styleContent.Reset()
			}
			if token.DataAtom == atom.Script {
				inScriptTag = true
				scriptContent.Reset()
			}
			_, err = writer.Write([]byte(token.String()))
		case html.TextToken:
			text := tokenizer.Text()
			if inStyleTag {
				styleContent.Write(text)
			} else if inScriptTag {
				scriptContent.Write(text)
			} else {
				_, err = writer.Write(text)
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			if token.DataAtom == atom.Style {
				inStyleTag = false
				modifiedText := processCSS(styleContent.String(), ci, sourceURL.Host, kubeapiAddr.String())
				_, err = writer.Write([]byte(modifiedText))
				if err != nil {
					logrus.Warnf("Write error: %s", err)
				}
			}
			if token.DataAtom == atom.Script {
				inScriptTag = false
				modifiedText := processJavaScript(scriptContent.String(), ci,
					sourceURL.Host, kubeapiAddr.String())
				_, err = writer.Write([]byte(modifiedText))
				if err != nil {
					logrus.Warnf("Write error: %s", err)
				}
			}
			_, err = writer.Write([]byte(token.String()))
		default:
			_, err = writer.Write(tokenizer.Raw())
		}
	}
	if err != io.EOF {
		return err
	}
	return nil
}

// rewriteResponse modifies an HTML response by updating absolute links referring
// to the original host to instead refer to the proxy transport.
func (t *RewritingTransport) rewriteResponse(
	req *http.Request, resp *http.Response) (*http.Response, error) {
	origBody := resp.Body
	defer origBody.Close()

	newContent := &bytes.Buffer{}
	var reader io.Reader = origBody
	var writer io.Writer = newContent
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("errorf making gzip reader: %v", err)
		}
		gzw := gzip.NewWriter(writer)
		defer gzw.Close()
		writer = gzw
	case "deflate":
		var err error
		reader = flate.NewReader(reader)
		flw, err := flate.NewWriter(writer, flate.BestCompression)
		if err != nil {
			return nil, fmt.Errorf("errorf making flate writer: %v", err)
		}
		defer func() {
			flw.Close()
			flw.Flush()
		}()
		writer = flw
	case "":
		// This is fine
	default:
		// Some encoding we don't understand-- don't try to parse this
		logrus.Errorf("Proxy encountered encoding %v for text/html; can't understand this so not fixing links.", encoding)
		return resp, nil
	}

	// These contain the host and protocol the browser used to access the ASP.
	// These contain the host and protocol the browser used to access the ASP.
	origHost := req.Header.Get("X-Forwarded-Host")
	origProto := req.Header.Get("X-Forwarded-Proto")
	sourceURL, err := url.Parse(origProto + "://" + origHost)
	if err != nil {
		logrus.Errorf("Proxy encountered error in parsing X-Forwarded headers: %v", err)
		return resp, nil
	}

	logrus.Debugf("sourceURL : %s", sourceURL)

	ci, err := extractCookieInfo(req)
	if err != nil {
		logrus.Errorf("Error retrieving request cookies: %v", err)
		return nil, err
	}

	kubeapiAddr, err := url.Parse(origProto + "://" + "kubernetes.default.svc")
	if err != nil {
		logrus.Errorf("Error creating kube API URL: %v", err)
	}

	logrus.Debugf("kubeapiAddr : %s", kubeapiAddr)
	err = rewriteHTML(reader, writer, ci, kubeapiAddr, sourceURL)
	if err != nil {
		logrus.Errorf("Failed to rewrite URLs: %v", err)
		return resp, err
	}

	resp.Body = io.NopCloser(newContent)
	// Update header node with new content-length
	// TODO: Remove any hash/signature headers here?
	resp.Header.Del("Content-Length")
	resp.ContentLength = int64(newContent.Len())

	// Fix the URLs in HTTP redirects.
	origRedirPath := resp.Header.Get("Location")
	if origRedirPath != "" {
		redirectURL, err := url.Parse(origRedirPath)
		if err != nil {
			logrus.Errorf("Error parsing redirect URL: %v", err)
			return resp, err
		}

		newRedirPath := rewriteURL(redirectURL, sourceURL, ci, kubeapiAddr)
		logrus.Debugf("Redirecting from %s to %s", origRedirPath, newRedirPath)
		resp.Header.Set("Location", newRedirPath)
	}

	return resp, err
}
