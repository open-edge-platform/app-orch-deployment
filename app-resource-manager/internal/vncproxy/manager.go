// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package vncproxy

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/wsproxy"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"net/http"
	"time"
)

var log = dazl.GetPackageLogger()

const (
	opaHostname = "localhost"
	opaScheme   = "http"
)

type Config struct {
	WSPort     int
	ConfigPath string
	FileBase   string
}

// Manager single point of entry for the provisioner
type Manager struct {
	Config Config
	router *mux.Router
}

// NewManager initializes the application manager
func NewManager(cfg Config) *Manager {
	m := &Manager{Config: cfg}
	m.initRouter(cfg.FileBase)
	return m
}

func (m *Manager) Run() {
	log.Info("Starting VNC-Proxy manager")
	if err := m.start(); err != nil {
		log.Fatalw("Unable to start manager", dazl.Error(err))
	}
}

func (m *Manager) start() error {
	// Create OPA client
	opaClient := opa.NewOPAClient(opaScheme, opaHostname)

	// Create session counters - one for counter per IP and the other one for counter per account
	ipSessionCounter := wsproxy.NewCounter(m.Config.ConfigPath, wsproxy.CounterTypeIP)
	accountSessionCounter := wsproxy.NewCounter(m.Config.ConfigPath, wsproxy.CounterTypeAccount)

	// Creates a SB handler to interact with Kubvirt API server

	admClient, err := adm.NewClient(m.Config.ConfigPath)
	if err != nil {
		return err
	}
	kubevirtManager := kubevirt.NewManager(m.Config.ConfigPath, admClient, true)
	m.router.HandleFunc(fmt.Sprintf("/%s/{project}/{app}/{cluster}/{vm}", kubevirt.VNCWebSocketPrefix),
		kubevirtManager.GetVNCWebSocketHandler(context.Background(),
			opaClient,
			ipSessionCounter,
			accountSessionCounter))
	http.HandleFunc(fmt.Sprintf("/%s/", kubevirt.VNCWebSocketPrefix), kubevirtManager.GetVNCWebSocketHandler(context.Background(), opaClient, ipSessionCounter, accountSessionCounter))

	listenAddress := fmt.Sprintf(":%d", m.Config.WSPort)
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           m.router,
		ReadHeaderTimeout: time.Minute * 5, // todo need to be refined
	}
	defer server.Close()
	log.Infow("Started WS server", dazl.String("address", listenAddress))
	err = server.ListenAndServe() // todo: need to be working with TLS
	if err != nil {
		log.Errorw("Error on WS server", dazl.String("error", err.Error()))
		return err
	}
	return nil
}

func (m *Manager) initRouter(filebase string) {
	m.router = mux.NewRouter()

	m.router.HandleFunc("/test", func(rw http.ResponseWriter, _ *http.Request) {
		if _, err := rw.Write([]byte("Ok\n")); err != nil {
			return
		}
	}).Methods("GET")

	m.router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		for k, v := range mux.Vars(req) {
			if len(v) == 0 {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err := rw.Write([]byte(fmt.Sprintf("%s is blank", k)))
				if err != nil {
					return
				}
				return
			}
		}
		rw.Header().Set("Content-Type", "text/html")
		http.ServeFile(rw, req, filebase+"/vnc-proxy-index.html")
	}).Methods("GET", "HEAD").Queries("project", "{project}", "app", "{app}", "cluster", "{cluster}", "vm", "{vm}")

	m.router.HandleFunc("/vnc-proxy-main.js", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(rw, req, filebase+"/vnc-proxy-main.js")
	}).Methods("GET", "HEAD")
	m.router.HandleFunc("/vnc-proxy-styles.css", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/css")
		http.ServeFile(rw, req, filebase+"/vnc-proxy-styles.css")
	}).Methods("GET", "HEAD")
	m.router.HandleFunc("/rfb.js", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(rw, req, filebase+"/rfb.js")
	}).Methods("GET", "HEAD")
	m.router.HandleFunc("/keycloak.min.js", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(rw, req, filebase+"/keycloak.min.js")
	}).Methods("GET", "HEAD")

	// VNC websocket handler is added in start()
}
