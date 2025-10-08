// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package skupperlib

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client"
	"github.com/open-edge-platform/orch-library/go/dazl"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/utils"
	skupperutils "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/utils/skupper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

var routerCreateOpts types.SiteConfigSpec
var connectorCreateOpts types.ConnectorCreateOptions

var log = dazl.GetLogger()

// SkupperInit initializes a Skupper site and creates a router
func SkupperInit(ctx context.Context, restConfig *rest.Config, ingressType string) error {

	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}

	// validate the site
	siteConfig, err := cli.SiteConfigInspect(ctx, nil)
	if err != nil {
		return err
	}

	switch ingressType {
	case skupperutils.IngressNone:
		routerCreateOpts.Ingress = types.IngressNoneString
	case skupperutils.IngressLoadBalancer:
		routerCreateOpts.Ingress = types.IngressLoadBalancerString
	default:
		routerCreateOpts.Ingress = types.IngressLoadBalancerString
	}

	routerCreateOpts.Labels = make(map[string]string)
	routerCreateOpts.SkupperNamespace = skupperutils.DefaultSkupperNamespace
	routerCreateOpts.EnableFlowCollector = false
	routerCreateOpts.EnableRestAPI = false
	routerCreateOpts.AuthMode = types.ConsoleAuthModeInternal
	//Enable cluster wide permissions in order to expose deployments/statefulsets in other namespaces
	routerCreateOpts.EnableClusterPermissions = true
	routerCreateOpts.EnableController = true
	routerCreateOpts.EnableSkupperEvents = true
	routerCreateOpts.EnableServiceSync = true
	routerCreateOpts.EnableConsole = false
	routerCreateOpts.CreateNetworkPolicy = false
	routerCreateOpts.EnableRestAPI = false
	routerCreateOpts.Annotations = make(map[string]string)
	routerCreateOpts.IngressAnnotations = make(map[string]string)

	routerCreateOpts.Routers = 0
	routerCreateOpts.RouterMode = string(types.TransportModeInterior)
	routerCreateOpts.Router.ServiceAnnotations = make(map[string]string)
	routerCreateOpts.Router.PodAnnotations = make(map[string]string)
	routerCreateOpts.Router.MaxFrameSize = types.RouterMaxFrameSizeDefault
	routerCreateOpts.Router.MaxSessionFrames = types.RouterMaxSessionFramesDefault
	routerCreateOpts.Router.DisableMutualTLS = false

	routerCreateOpts.Controller.ServiceAnnotations = make(map[string]string)
	routerCreateOpts.Controller.PodAnnotations = make(map[string]string)

	routerCreateOpts.PrometheusServer.PodAnnotations = make(map[string]string)

	routerCreateOpts.FlowCollector.FlowRecordTtl = 0
	//routerCreateOpts.SiteTtl = 0
	routerCreateOpts.RunAsUser = 0
	routerCreateOpts.RunAsGroup = 0

	//unsure if this is needed
	//routerCreateOpts.Platform

	// Initialize Skupper site
	if siteConfig == nil {
		siteConfig, err = cli.SiteConfigCreate(ctx, routerCreateOpts)
		if err != nil {
			return err
		}
	} else {
		updated, err := cli.SiteConfigUpdate(ctx, routerCreateOpts)
		if err != nil {
			return fmt.Errorf("error while trying to update router configuration: %s", err)
		}
		log.Infof("Site config is updated %s", updated)
	}

	// Create Skupper router
	err = cli.RouterCreate(ctx, *siteConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper router: %w", err)
	}

	return nil
}

func SkupperDelete(ctx context.Context, restConfig *rest.Config) error {
	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}
	gateways, err := cli.GatewayList(ctx)
	for _, gateway := range gateways {
		cli.GatewayRemove(ctx, gateway.Name)
	}
	err = cli.SiteConfigRemove(ctx)
	if err != nil {
		err = cli.RouterRemove(ctx)
	}
	if err != nil {
		return err
	} else {
		log.Infof("Skupper is now removed from %s", cli.GetNamespace())
	}
	return nil
}

// SkupperTokenCreate creates a Skupper token
func SkupperTokenCreate(ctx context.Context, restConfig *rest.Config, secretName string) (*corev1.Secret, error) {

	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Skupper client: %w", err)
	}

	password := utils.RandomId(24)
	expiry := 15 * time.Minute
	uses := 1

	// this should be enhance to return the token itself
	secret, _, err := cli.TokenClaimCreate(ctx, secretName, []byte(password), expiry, uses)
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create Skupper token: %w", err)
	}
	if err != nil && errors.IsAlreadyExists(err) {
		log.Infof("Token %s already exists", secretName)
		clientSet, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, err
		}
		err = clientSet.CoreV1().Secrets(skupperutils.DefaultSkupperNamespace).Delete(ctx, secretName, metav1.DeleteOptions{})
		if err != nil {
			log.Infof("Failed to delete secret %s", secretName)
			return nil, err
		}
		secret, _, err = cli.TokenClaimCreate(ctx, secretName, []byte(password), expiry, uses)
		if err != nil && !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create Skupper token: %w", err)
		}

		return secret, nil
	}

	return secret, nil
}

// SkupperLinkCreate creates a Skupper link between sites
func SkupperLinkCreate(ctx context.Context, restConfig *rest.Config, secret *corev1.Secret, linkName string) error {
	connectorCreateOpts.Secret = secret
	/*if secret.ObjectMeta.Annotations != nil && !costFlag.Changed {
		if costStr, ok := secret.ObjectMeta.Annotations[types.TokenCost]; ok {
			if cost, err := strconv.Atoi(costStr); err == nil {
				connectorCreateOpts.Cost = int32(cost)
			}
		}
	}*/

	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}

	connectorCreateOpts.SkupperNamespace = skupperutils.DefaultSkupperNamespace
	connectorCreateOpts.Name = linkName
	connectorCreateOpts.Cost = 1
	secret2, err := cli.ConnectorCreateSecretFromData(ctx, connectorCreateOpts)
	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	} else {
		log.Infof("Site configured to link to %s (name=%s)",
			secret2.ObjectMeta.Annotations[types.ClaimUrlAnnotationKey],
			secret2.ObjectMeta.Name)
	}

	return nil
}

func SkupperLinkDelete(ctx context.Context, sourceClusterRESTConfig *rest.Config, destClusterRESTConfig *rest.Config, linkName string) error {
	// Create a Skupper client
	connectorRemoveOpts := types.ConnectorRemoveOptions{}
	connectorRemoveOpts.SkupperNamespace = skupperutils.DefaultSkupperNamespace
	connectorRemoveOpts.Name = linkName
	connectorRemoveOpts.ForceCurrent = false

	cli, err := client.NewClientWithRESTConfig(destClusterRESTConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}

	err = cli.ConnectorRemove(ctx, connectorRemoveOpts)
	if err != nil {
		return fmt.Errorf("failed to remove link: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(sourceClusterRESTConfig)
	if err != nil {
		return err
	}
	err = clientSet.CoreV1().Secrets(skupperutils.DefaultSkupperNamespace).Delete(ctx, linkName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func SkupperUnexposeService(ctx context.Context, restConfig *rest.Config, targetName string, targetType string, address string) error {
	if targetType == "deployment" {
		return fmt.Errorf("exposing target type deployment isn't supported")
	}
	if address == "" && targetType == "service" {
		return fmt.Errorf("--address option is required for target type 'service'")
	}
	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}
	// we dont use target-namespace with targetType == "service"

	err = cli.ServiceInterfaceUnbind(ctx, targetType, targetName, address, true, "")
	if err == nil {
		log.Infof("%s %s unexposed", targetType, targetName)
	} else {
		return fmt.Errorf("unable to unbind skupper service: %w", err)
	}
	return nil
}

// SkupperExposeService exposes a service using Skupper
func SkupperExposeService(ctx context.Context, restConfig *rest.Config, targetName string, targetType string, address string, ports []int) error {
	if targetType == "deployment" {
		return fmt.Errorf("exposing targetType deployment isn't supported")
	}
	if address == "" && targetType == "service" {
		return fmt.Errorf("--address option is required for target type 'service'")
	}
	// Create a Skupper client
	cli, err := client.NewClientWithRESTConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Skupper client: %w", err)
	}

	service, err := cli.ServiceInterfaceInspect(ctx, address)
	if err != nil {
		log.Warn(err)
		return err
	}

	var policy *client.PolicyAPIClient
	policy = client.NewPolicyValidatorAPI(cli)
	res, err := policy.Expose(targetType, targetName)
	if err != nil {
		log.Warn(err)
		return err
	}
	if !res.Allowed {
		return res.Err()
	}

	if service == nil {
		res, err := policy.Service(address)
		if err != nil {
			return err
		}
		if !res.Allowed {
			return res.Err()
		}
		service = &types.ServiceInterface{
			Address:                  address,
			Ports:                    ports,
			Protocol:                 "tcp",
			TlsCredentials:           "",
			TlsCertAuthority:         "",
			PublishNotReadyAddresses: false,
			BridgeImage:              "",
		}
		err = service.SetIngressMode("")
		if err != nil {
			return err

		}
	}

	// service may exist from remote origin
	service.Origin = ""

	targetPorts := map[int]int{}
	for _, port := range ports {
		targetPorts[port] = port
	}

	err = cli.ServiceInterfaceBind(ctx, service, targetType, targetName, targetPorts, "")
	if errors.IsNotFound(err) {
		return SkupperNotInstalledError(cli.GetNamespace())
	} else if err != nil {
		return fmt.Errorf("unable to create skupper service: %w", err)
	}
	log.Infof("%s %s exposed as %s\n", targetType, targetName, address)
	return nil
}

func SkupperNotInstalledError(namespace string) error {
	return fmt.Errorf("Skupper is not installed in Namespace: '" + namespace + "`")

}
