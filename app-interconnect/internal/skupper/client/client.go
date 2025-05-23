package client

import (
	skupperutils "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/utils/skupper"
	"time"

	openshiftapps "github.com/openshift/client-go/apps/clientset/versioned"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/domain"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/kube"
)

var defaultRetry = wait.Backoff{
	Steps:    100,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// A VAN Client manages orchestration and communications with the network components
type VanClient struct {
	Namespace       string
	KubeClient      kubernetes.Interface
	RouteClient     *routev1client.RouteV1Client
	OCAppsClient    openshiftapps.Interface
	RestConfig      *restclient.Config
	DynamicClient   dynamic.Interface
	DiscoveryClient *discovery.DiscoveryClient
	LinkHandler     domain.LinkHandler
}

func (cli *VanClient) GetNamespace() string {
	return cli.Namespace
}

func (cli *VanClient) GetKubeClient() kubernetes.Interface {
	return cli.KubeClient
}

func (cli *VanClient) GetDynamicClient() dynamic.Interface {
	return cli.DynamicClient
}

func (cli *VanClient) GetDiscoveryClient() *discovery.DiscoveryClient {
	return cli.DiscoveryClient
}

func (cli *VanClient) GetRouteClient() *routev1client.RouteV1Client {
	return cli.RouteClient
}

func (cli *VanClient) GetVersion(component string, name string) string {
	return kube.GetComponentVersion(cli.Namespace, cli.KubeClient, component, name)
}

func NewClientWithRESTConfig(restConfig *restclient.Config) (*VanClient, error) {
	c := &VanClient{}

	restConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Version: "v1"}
	restConfig.APIPath = "/api"
	restConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	c.RestConfig = restConfig

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return c, err
	}
	c.KubeClient = kubeClient

	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	resources, err := dc.ServerResourcesForGroupVersion("route.openshift.io/v1")
	if err == nil && len(resources.APIResources) > 0 {
		c.RouteClient, err = routev1client.NewForConfig(restConfig)
		if err != nil {
			return c, err
		}
	}
	resources, err = dc.ServerResourcesForGroupVersion("apps.openshift.io/v1")
	if err == nil && len(resources.APIResources) > 0 {
		c.OCAppsClient, err = openshiftapps.NewForConfig(restConfig)
		if err != nil {
			return c, err
		}
	}

	c.DiscoveryClient = dc
	c.Namespace = skupperutils.DefaultSkupperNamespace

	c.DynamicClient, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		return c, err
	}

	return c, nil
}

func NewClient(namespace string, context string, kubeConfigPath string) (*VanClient, error) {
	c := &VanClient{}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeConfigPath != "" {
		loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}
	}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		},
	)
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return c, err
	}
	restconfig.ContentConfig.GroupVersion = &schema.GroupVersion{Version: "v1"}
	restconfig.APIPath = "/api"
	restconfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	c.RestConfig = restconfig
	c.KubeClient, err = kubernetes.NewForConfig(restconfig)
	if err != nil {
		return c, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restconfig)
	resources, err := dc.ServerResourcesForGroupVersion("route.openshift.io/v1")
	if err == nil && len(resources.APIResources) > 0 {
		c.RouteClient, err = routev1client.NewForConfig(restconfig)
		if err != nil {
			return c, err
		}
	}
	resources, err = dc.ServerResourcesForGroupVersion("apps.openshift.io/v1")
	if err == nil && len(resources.APIResources) > 0 {
		c.OCAppsClient, err = openshiftapps.NewForConfig(restconfig)
		if err != nil {
			return c, err
		}
	}

	c.DiscoveryClient = dc

	if namespace == "" {
		c.Namespace, _, err = kubeconfig.Namespace()
		if err != nil {
			return c, err
		}
	} else {
		c.Namespace = namespace
	}
	c.DynamicClient, err = dynamic.NewForConfig(restconfig)
	if err != nil {
		return c, err
	}

	return c, nil
}

func (cli *VanClient) GetIngressDefault() string {
	if cli.RouteClient == nil {
		return types.IngressLoadBalancerString
	}
	return types.IngressRouteString
}
