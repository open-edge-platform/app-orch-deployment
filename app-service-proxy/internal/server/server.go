// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/rbac"
	"github.com/sirupsen/logrus"

	//Blank import
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/admclient"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/middleware"
)

const (
	OIDCServerURL                = "OIDC_SERVER_URL"
	PathPrefix                   = "/app-service-proxy"
	MaxBodySizeBytesLimit        = "MAX_BODY_SIZE_BYTES_LIMIT"
	DefaultMaxBodySizeBytesLimit = 100 // mega-bytes
)

type CookieInfo struct {
	projectID string
	cluster   string
	namespace string
	service   string
}

type Server struct {
	addr         string
	ccgAddress   string
	router       *mux.Router
	server       *http.Server
	authNenabled bool
	authZenabled bool
	authenticate func(req *http.Request) error
	authorize    func(req *http.Request, projectID string) error
	admClient    admclient.ADMClient
}

func extractCookieInfo(req *http.Request) (*CookieInfo, error) {
	ci := &CookieInfo{}
	var err error

	cookie, err := req.Cookie("app-service-proxy-project")
	if err != nil {
		logrus.Errorf("Error retrieving app-service-proxy-project cookie: %v", err)
		return nil, err
	}
	ci.projectID = cookie.Value

	cookie, err = req.Cookie("app-service-proxy-cluster")
	if err != nil {
		logrus.Errorf("Error retrieving app-service-proxy-cluster cookie: %v", err)
		return nil, err
	}
	ci.cluster = cookie.Value

	cookie, err = req.Cookie("app-service-proxy-namespace")
	if err != nil {
		logrus.Errorf("Error retrieving app-service-proxy-namespace cookie: %v", err)
		return nil, err
	}
	ci.namespace = cookie.Value

	cookie, err = req.Cookie("app-service-proxy-service")
	if err != nil {
		logrus.Errorf("Error retrieving app-service-proxy-service cookie: %v", err)
		return nil, err
	}
	ci.service = cookie.Value

	return ci, nil
}

func sanitizeCookieHeader(cookieHeader string) string {
	cookieHeader = strings.TrimPrefix(cookieHeader, "[\"")
	cookieHeader = strings.TrimSuffix(cookieHeader, "\"]")
	sanitizedParts := make([]string, 0)
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "app-service-proxy-token") {
			sanitizedParts = append(sanitizedParts, part)
		}
	}
	return strings.Join(sanitizedParts, "; ")
}

func NewServer(addr string) (*Server, error) {
	// To allow connections to be reused, we need to set the following parameters.
	// By default, the http.DefaultTransport will set MaxIdleConnsPerHost to 2.
	// All outgoing requests from ASP to a CCG are to the same host, hence we
	// increase this to prevent connections from constantly being opened and closed.
	http.DefaultTransport.(*http.Transport).DisableKeepAlives = false
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 64

	a := &Server{
		addr: addr,
	}

	if a.ccgAddress = os.Getenv("CCG_ADDRESS"); a.ccgAddress == "" {
		return nil, fmt.Errorf("CCG_ADDRESS is not set")
	}

	if l, ok := os.LookupEnv("ASP_LOG_LEVEL"); ok {
		level, err := logrus.ParseLevel(l)
		if err != nil {
			return nil, fmt.Errorf("ASP_LOG_LEVEL is not set to a valid logrus level")
		}
		logrus.SetLevel(level)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	a.initAuth()
	a.initRouter()

	return a, nil
}

func (a *Server) Run() error {
	var err error
	if err = a.initAdmClient(); err != nil {
		return nil
	}
	logrus.Infof("Listening on %s", a.addr)
	a.server = &http.Server{Addr: a.addr, Handler: a.router, ReadTimeout: 10 * time.Second}
	return a.server.ListenAndServe()
}

func (a *Server) Close() error {
	logrus.Info("Closing server")
	return a.server.Close()
}

func (a *Server) ServicesProxy(rw http.ResponseWriter, req *http.Request) {
	logrus.Infof("ServicesProxy default handler for path %s", req.URL.Path)

	ci, err := extractCookieInfo(req)
	if err != nil {
		if err == http.ErrNoCookie {
			logrus.Errorf("Cookie not present - redirecting: %v", err)
			rw.Header().Set("X-Reason", err.Error())
			http.Redirect(rw, req, "/app-service-proxy-index.html", http.StatusFound)
			return
		}
		logrus.Errorf("Error extracting cookie info: %v", err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	logrus.Infof("CookieInfo: %+v", ci)

	err = a.isAllowed(req, ci.projectID)
	if err != nil {
		logrus.Warnf("Failed to authenticate/authorize request: %v", err)
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	// Parse the target URL. Is always "app-service-proxy.kind.internal" which
	// maps to "kubernetes.default.svc.cluster.local" on the edge-node.
	target, err := url.Parse("http" + "://" + a.ccgAddress)
	if err != nil {
		logrus.Errorf("Error parsing URL %s: %s", target, err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	logrus.Debugf("target : %s", target)
	timeout := req.URL.Query().Get("timeout")
	timeoutVal := 15 * time.Second
	if timeout != "" {
		val, err := strconv.Atoi(timeout)
		if err == nil {
			timeoutVal = time.Duration(val) * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutVal)
	defer cancel()
	clusterInfraName, err := a.admClient.GetClusterInfraName(ctx, ci.cluster,
		ci.projectID)
	if err != nil {
		logrus.Warnf("Failed to get Capi Infra name %v", err)
		return
	}

	// Create proxy and set the Transport rancher/remoteDialer client
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &RewritingTransport{}
	newPath := fmt.Sprintf("/kubernetes/%s-%s/api/v1/namespaces/%s/services/%s/proxy%s",
		ci.projectID, clusterInfraName, ci.namespace, ci.service, req.URL.Path)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		for _, cookie := range req.Cookies() {
			if strings.HasPrefix(cookie.Name, "app-service-proxy-token") {
				logrus.Debugf("Resetting cookie: %s", cookie.Name)
				req.AddCookie(&http.Cookie{Name: cookie.Name, Value: "", MaxAge: -1})
			}
		}
		req.Header.Set("Cookie", sanitizeCookieHeader(req.Header.Get("Cookie")))
		req.URL.Path = newPath
		req.Host = a.ccgAddress
		existingHeader := req.Header.Get("Authorization")
		if existingHeader != "" {
			req.Header.Add("X-App-Authorization", existingHeader)
			logrus.Infof("Moved App Authorization Header to X-App-Authorization: %s", existingHeader)
			req.Header.Del("Authorization") // Remove the incoming Authorization header or else they will be merged
		}

		req.Header.Add("ActiveProjectID", ci.projectID)
	}

	proxy.ServeHTTP(rw, req)
}

func (a *Server) isAllowed(req *http.Request, projectID string) error {
	if a.authNenabled {
		err := a.authenticate(req)
		if err != nil {
			logrus.Warnf("Authentication failed: %v", err)
			return err
		}
	}
	if a.authZenabled {
		err := a.authorize(req, projectID)
		if err != nil {
			logrus.Warnf("Authorization failed: %v", err)
			return err
		}
	}
	return nil
}

func (a *Server) maxBodySizeBytesLimit() int64 {
	maxBodySizeBytesLimit, err := strconv.Atoi(os.Getenv(MaxBodySizeBytesLimit))
	if err != nil {
		maxBodySizeBytesLimit = DefaultMaxBodySizeBytesLimit
	}
	return int64(maxBodySizeBytesLimit << 20)
}

func (a *Server) initRouter() {
	a.router = mux.NewRouter()
	a.router.HandleFunc("/app-service-proxy-test", func(rw http.ResponseWriter, _ *http.Request) {
		if _, err := rw.Write([]byte("Ok\n")); err != nil {
			return
		}
	}).Methods("GET")
	a.router.HandleFunc("/", a.ServicesProxy)

	a.router.HandleFunc("/app-service-proxy-index.html", func(rw http.ResponseWriter, req *http.Request) {
		for k, v := range mux.Vars(req) {
			if len(v) == 0 {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = rw.Write([]byte(fmt.Sprintf("%s is empty", k)))
				return
			}
		}
		rw.Header().Set("Content-Type", "text/html")
		http.ServeFile(rw, req, "web-login/app-service-proxy-index.html")
	}).Methods("GET").Queries("project", "{project}", "cluster", "{cluster}", "namespace", "{namespace}", "service", "{service}")

	a.router.HandleFunc("/app-service-proxy-index.html", func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = rw.Write([]byte("Missing some query parameter: project, cluster, namespace, service"))
	}).Methods("GET")

	a.router.HandleFunc("/app-service-proxy-keycloak.min.js", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(rw, req, "web-login/app-service-proxy-keycloak.min.js")
	}).Methods("GET")
	a.router.HandleFunc("/app-service-proxy-main.js", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(rw, req, "web-login/app-service-proxy-main.js")
	}).Methods("GET")
	a.router.HandleFunc("/app-service-proxy-styles.css", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/css")
		http.ServeFile(rw, req, "web-login/app-service-proxy-styles.css")
	}).Methods("GET")

	maxBodySizeBytesLimit := a.maxBodySizeBytesLimit()
	a.router.Use(middleware.SizeLimitMiddleware(maxBodySizeBytesLimit))
	a.router.PathPrefix("/").HandlerFunc(a.ServicesProxy) // Must be registered last, default handler.
}

func (a *Server) initAuth() {
	// Authentication
	if oidcURL := os.Getenv(OIDCServerURL); oidcURL != "" {
		a.authNenabled = true
		logrus.Infof("Authentication is enabled, OIDC server address is %s", oidcURL)
	} else {
		logrus.Warnf("Authentication is disabled")
	}
	a.authenticate = rbac.AuthenticateFunc

	// Authorization
	if os.Getenv("OPA_ENABLED") == "true" {
		a.authZenabled = true
		logrus.Infof("Authorization is enabled")
	} else {
		logrus.Warnf("Authorization is disabled")
	}
	a.authorize = rbac.AuthorizeFunc
}

func (a *Server) initAdmClient() error {
	var err error
	if a.admClient, err = admclient.NewClient(); err != nil {
		return err
	}
	return nil
}
