// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/pkg/restClient"
)

var portForwardCmd = make(map[string]*exec.Cmd)

const (
	PortForwardServiceNamespace = "orch-app"
	AdmPortForwardService       = "svc/app-deployment-api-rest-proxy"
	AdmPortForwardLocal         = "8081"
	PortForwardAddress          = "0.0.0.0"
	AdmPortForwardRemote        = "8081"
)

func PortForward(scenario string, portForwardCmd map[string]*exec.Cmd) (map[string]*exec.Cmd, error) {
	service := AdmPortForwardService
	localPort := AdmPortForwardLocal
	remotePort := AdmPortForwardRemote

	portPort := fmt.Sprintf("%s:%s", remotePort, localPort)

	fmt.Printf("%s port-forward with service %s, ports %s\n", scenario, service, portPort)

	// Check if the port-forward command is already running
	if cmd, exists := portForwardCmd[scenario]; exists && cmd != nil && cmd.Process != nil {
		fmt.Printf("%s port-forward command already running\n", scenario)
		return portForwardCmd, nil
	}

	cmd := exec.Command("kubectl", "port-forward", "-n", PortForwardServiceNamespace, service, portPort, "--address", PortForwardAddress)
	if cmd == nil {
		return portForwardCmd, fmt.Errorf("failed to create kubectl command")
	}

	err := cmd.Start()
	if err != nil {
		return portForwardCmd, fmt.Errorf("failed to start kubectl command: %v", err)
	}

	time.Sleep(5 * time.Second) // Give some time for port-forwarding to establish
	portForwardCmd[scenario] = cmd

	return portForwardCmd, err
}

func KillPortForward(scenario string, portForwardCmd map[string]*exec.Cmd) error {
	cmd := portForwardCmd[scenario]
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

func BringUpPortForward() (map[string]*exec.Cmd, error) {
	portForwardCmd, err := PortForward("adm", portForwardCmd)
	if err != nil {
		return nil, fmt.Errorf("error: %v", err)
	}

	return portForwardCmd, nil
}

func TearDownPortForward(portForwardCmd map[string]*exec.Cmd) {
	scenario := "adm"
	err := KillPortForward(scenario, portForwardCmd)
	if err == nil {
		fmt.Printf("%s port-forward process killed\n", scenario)
	}
}

func CallMethod(url, verb, token, projectID string) (*http.Response, error) {
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return nil, err
	}

	AddRestAuthHeader(req, token, projectID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, err
}

func CreateClient(restServerURL, token, projectID string) (*restClient.ClientWithResponses, error) {
	armClient, err := restClient.NewClientWithResponses(restServerURL, restClient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return armClient, err
}
