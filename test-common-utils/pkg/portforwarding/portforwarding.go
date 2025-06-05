// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package portforwarding

import (
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/types"
	"net/http"
	"os/exec"
	"time"
)

var portForwardCmd = make(map[string]*exec.Cmd)

func PortForward(scenario string, portForwardCmd map[string]*exec.Cmd) (map[string]*exec.Cmd, error) {

	service := types.AdmPortForwardService
	localPort := types.AdmPortForwardLocal
	remotePort := types.AdmPortForwardRemote

	if scenario == "arm" {
		service = types.ArmPortForwardService
		localPort = types.ArmPortForwardLocal
		remotePort = types.ArmPortForwardRemote
	}

	portPort := fmt.Sprintf("%s:%s", remotePort, localPort)

	fmt.Printf("%s port-forward with service %s, ports %s\n", scenario, service, portPort)

	// Check if the port-forward command is already running
	if cmd, exists := portForwardCmd[scenario]; exists && cmd != nil && cmd.Process != nil {
		fmt.Printf("%s port-forward command already running\n", scenario)
		return portForwardCmd, nil
	}

	// #nosec G204 -- Arguments are controlled and validated within the application context.
	cmd := exec.Command("kubectl", "port-forward", "-n", types.PortForwardServiceNamespace, service, portPort, "--address", types.PortForwardAddress)
	if cmd == nil {
		return portForwardCmd, fmt.Errorf("failed to create kubectl command")
	}

	err := cmd.Start()
	if err != nil {
		return portForwardCmd, fmt.Errorf("failed to start kubectl command: %v", err)
	}

	time.Sleep(10 * time.Second) // Give some time for port-forwarding to establish
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

func StartPortForwarding() (map[string]*exec.Cmd, error) {
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

func CallMethod(url, verb, token, projectID string) (*http.Response, error) {
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return nil, err
	}

	auth.AddRestAuthHeader(req, token, projectID)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, err
}
