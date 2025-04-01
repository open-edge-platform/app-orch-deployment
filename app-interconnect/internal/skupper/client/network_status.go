package client

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	k8s "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/kube"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/network"
	"strings"
)

func (cli *VanClient) NetworkStatus(ctx context.Context) (*network.NetworkStatusInfo, error) {

	//Checking if the router has been deployed
	_, err := k8s.GetDeployment(types.TransportDeploymentName, cli.Namespace, cli.KubeClient)
	if err != nil {
		return nil, fmt.Errorf("Skupper is not installed: %s", err)
	}

	configmap, err := k8s.GetConfigMap(types.NetworkStatusConfigMapName, cli.Namespace, cli.KubeClient)
	if err != nil && strings.Contains(err.Error(), "\"skupper-network-status\" not found") {
		return nil, fmt.Errorf("status not ready")
	} else if err != nil {
		return nil, err
	}

	vanInfo, err := network.UnmarshalSkupperStatus(configmap.Data)
	if err != nil && strings.Contains(err.Error(), "unexpected end of JSON input") {
		return nil, fmt.Errorf("status not ready")
	} else if err != nil {
		return nil, err
	}

	return vanInfo, nil
}
