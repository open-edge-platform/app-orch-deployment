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

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/pkg/restClient/v2"
)

var portForwardCmd = make(map[string]*exec.Cmd)

const (
	PortForwardServiceNamespace = "orch-app"
	AdmPortForwardService       = "svc/app-deployment-api-rest-proxy"
	ArmPortForwardService       = "svc/app-resource-manager-rest-proxy"
	AdmPortForwardLocal         = "8081"
	ArmPortForwardLocal         = "8081"
	PortForwardAddress          = "0.0.0.0"
	AdmPortForwardRemote        = "8081"
	ArmPortForwardRemote        = "8082"
	AspPortForwardService       = "svc/app-service-proxy"
	AspPortForwardLocal         = "8123"
	AspPortForwardRemote        = "8123"
)

func PortForward(scenario string, portForwardCmd map[string]*exec.Cmd) (map[string]*exec.Cmd, error) {
	service := AdmPortForwardService
	localPort := AdmPortForwardLocal
	remotePort := AdmPortForwardRemote

	if scenario == "arm" {
		service = ArmPortForwardService
		localPort = ArmPortForwardLocal
		remotePort = ArmPortForwardRemote
	}

	if scenario == "asp" {
		service = AspPortForwardService
		localPort = AspPortForwardLocal
		remotePort = AspPortForwardRemote
	}

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

	portForwardCmd, err = PortForward("arm", portForwardCmd)
	if err != nil {
		return nil, fmt.Errorf("error: %v", err)
	}
	return portForwardCmd, nil
}

func BringUpASPPortForward() (map[string]*exec.Cmd, error) {
	portForwardCmd, err := PortForward("asp", portForwardCmd)
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

	scenario = "arm"
	err = KillPortForward(scenario, portForwardCmd)
	if err == nil {
		fmt.Printf("%s port-forward process killed\n", scenario)
	}
}

func TearDownASPPortForward(portForwardCmd map[string]*exec.Cmd) {
	scenario := "asp"
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

func CreateArmClient(restServerURL, token, projectID string) (*restClient.ClientWithResponses, error) {
	armClient, err := restClient.NewClientWithResponses(restServerURL, restClient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		AddRestAuthHeader(req, token, projectID)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return armClient, err
}
