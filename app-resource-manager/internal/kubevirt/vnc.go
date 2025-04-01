// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"kubevirt.io/client-go/kubecli"
	"net/http"
	"strings"
)

func (m *manager) getVNCStream(ctx context.Context, client kubecli.KubevirtClient, appID string, vmID string) (kubecli.StreamInterface, error) {
	// app ID and VM ID is valid
	vm, err := m.getKubevirtVirtualMachine(ctx, client, appID, vmID)
	if err != nil {
		return nil, err
	}

	return client.VirtualMachineInstance(vm.Namespace).VNC(vm.Name)
}

func (m *manager) getVNCWebSocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
}

func newVNCPath(path string) (*vncPath, error) {
	// validate vnc path
	pathElems := strings.Split(path, "/")
	if len(pathElems) != 6 || pathElems[0] != "" || pathElems[1] != VNCWebSocketPrefix {
		return nil, errors.NewInvalid("wrong VNC path format: path-%s", path)
	}

	return &vncPath{
		path:      path,
		appID:     pathElems[3],
		clusterID: pathElems[4],
		vmID:      pathElems[5],
		projectID: pathElems[2],
	}, nil
}

type vncPath struct {
	path      string
	appID     string
	clusterID string
	vmID      string
	projectID string
}

type WebSocketWriter struct {
	conn *websocket.Conn
}

func (w WebSocketWriter) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type WebSocketReader struct {
	conn *websocket.Conn
}

func (r WebSocketReader) Read(p []byte) (n int, err error) {
	_, reader, err := r.conn.NextReader()
	if err != nil {
		return 0, err
	}
	return reader.Read(p)
}
