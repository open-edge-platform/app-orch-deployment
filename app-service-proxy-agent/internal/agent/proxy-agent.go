// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/cert"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/token"

	"github.com/gorilla/websocket"
	"github.com/rancher/remotedialer"
	"github.com/sirupsen/logrus"
)

type ServiceProxyAgent struct {
	closed             chan struct{}
	serverURL          string
	insecureSkipVerify bool
	agentID            string
	agentToken         string
}

const (
	TunnelID = "X-Tunnel-ID"        // #nosec G101
	Token    = "X-API-Tunnel-Token" // #nosec G101
)

func NewServiceProxyAgent() (*ServiceProxyAgent, error) {
	logrus.SetLevel(logrus.DebugLevel)

	serverURL, ok := os.LookupEnv("PROXY_SERVER_URL")
	if !ok || serverURL == "" {
		return nil, fmt.Errorf("PROXY_SERVER_URL is not set")
	}

	// Set insecure to false unless it's set to true explicitly
	insecure := true
	skip, ok := os.LookupEnv("INSECURE_SKIP_VERIFY")
	if !ok || skip == "" || skip == "false" {
		insecure = false
	}

	id, ok := os.LookupEnv("AGENT_ID")
	if !ok || id == "" {
		return nil, fmt.Errorf("AGENT_ID is not set")
	}

	token, err := os.ReadFile(token.TokenFilePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read token(%v)", err)
	}

	remotedialer.PrintTunnelData = false
	if os.Getenv("CATTLE_TUNNEL_DATA_DEBUG") == "true" {
		remotedialer.PrintTunnelData = true
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

	agent := &ServiceProxyAgent{
		closed:             make(chan struct{}),
		serverURL:          fmt.Sprintf("%s/connect", serverURL),
		insecureSkipVerify: insecure,
		agentID:            id,
		agentToken:         string(token),
	}

	return agent, nil
}

func (s *ServiceProxyAgent) Run(ctx context.Context) {
	logrus.Infof("ServiceProxy agent is starting")

	headers := http.Header{Token: {s.agentToken}, TunnelID: {s.agentID}}
	dialer := websocket.DefaultDialer

	dialer.TLSClientConfig = cert.GetTLSConfigs(s.insecureSkipVerify)

	connAuthorizer := func(proto, address string) bool {
		switch proto {
		case "tcp":
			return true
		case "unix":
			return address == "/var/run/docker.sock"
		case "npipe":
			return address == "//./pipe/docker_engine"
		}
		return false
	}

	onConnect := func(_ context.Context, _ *remotedialer.Session) error {
		// Do nothing on successful connection now
		// Periodic checks can be added here later
		return nil
	}

	if err := remotedialer.ClientConnect(ctx,
		s.serverURL,
		headers,
		dialer,
		connAuthorizer,
		onConnect); err != nil {
		errMsg := fmt.Errorf("Unable to connect to %s (%s): %v", s.serverURL, s.agentID, err)
		logrus.Error(errMsg)
		panic(errMsg)
	}
}

func (s *ServiceProxyAgent) Stop() {
	close(s.closed)
}
