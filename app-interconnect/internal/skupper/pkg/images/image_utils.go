package images

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	corev1 "k8s.io/api/core/v1"
)

const (
	RouterImageEnvKey                 string = "QDROUTERD_IMAGE"
	ServiceControllerImageEnvKey      string = "SKUPPER_SERVICE_CONTROLLER_IMAGE"
	ControllerPodmanImageEnvKey       string = "SKUPPER_CONTROLLER_PODMAN_IMAGE"
	ConfigSyncImageEnvKey             string = "SKUPPER_CONFIG_SYNC_IMAGE"
	FlowCollectorImageEnvKey          string = "SKUPPER_FLOW_COLLECTOR_IMAGE"
	PrometheusServerImageEnvKey       string = "PROMETHEUS_SERVER_IMAGE"
	OauthProxyImageEnvKey             string = "OAUTH_PROXY_IMAGE"
	RouterPullPolicyEnvKey            string = "QDROUTERD_IMAGE_PULL_POLICY"
	ServiceControllerPullPolicyEnvKey string = "SKUPPER_SERVICE_CONTROLLER_IMAGE_PULL_POLICY"
	ControllerPodmanPullPolicyEnvKey  string = "SKUPPER_CONTROLLER_PODMAN_IMAGE_PULL_POLICY"
	ConfigSyncPullPolicyEnvKey        string = "SKUPPER_CONFIG_SYNC_IMAGE_PULL_POLICY"
	FlowCollectorPullPolicyEnvKey     string = "SKUPPER_FLOW_COLLECTOR_IMAGE_PULL_POLICY"
	PrometheusServerPullPolicyEnvKey  string = "PROMETHEUS_SERVER_IMAGE_PULL_POLICY"
	OauthProxyPullPolicyEnvKey        string = "OAUTH_PROXY_IMAGE_PULL_POLICY"
	SkupperImageRegistryEnvKey        string = "SKUPPER_IMAGE_REGISTRY"
	PrometheusImageRegistryEnvKey     string = "PROMETHEUS_IMAGE_REGISTRY"
	OauthProxyRegistryEnvKey          string = "OAUTH_PROXY_IMAGE_REGISTRY"
)

func getPullPolicy(key string) string {
	policy := os.Getenv(key)
	if policy == "" {
		policy = string(corev1.PullAlways)
	}
	return policy
}

func GetRouterImageName() string {
	image := os.Getenv(RouterImageEnvKey)
	if image == "" {
		imageRegistry := GetImageRegistry()
		return strings.Join([]string{imageRegistry, RouterImageName}, "/")

	} else {
		return image
	}
}

func GetRouterImagePullPolicy() string {
	return getPullPolicy(RouterPullPolicyEnvKey)
}

func GetRouterImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetRouterImageName(),
		PullPolicy: GetRouterImagePullPolicy(),
	}
}

func AddRouterImageOverrideToEnv(env []corev1.EnvVar) []corev1.EnvVar {
	result := env
	image := os.Getenv(RouterImageEnvKey)
	if image != "" {
		result = append(result, corev1.EnvVar{Name: RouterImageEnvKey, Value: image})
	}
	policy := os.Getenv(RouterPullPolicyEnvKey)
	if policy != "" {
		result = append(result, corev1.EnvVar{Name: RouterPullPolicyEnvKey, Value: policy})
	}
	return result
}

func GetServiceControllerImageName() string {
	image := os.Getenv(ServiceControllerImageEnvKey)
	if image == "" {
		imageRegistry := GetImageRegistry()
		return strings.Join([]string{imageRegistry, ServiceControllerImageName}, "/")
	} else {
		return image
	}
}

func GetServiceControllerImagePullPolicy() string {
	return getPullPolicy(ServiceControllerPullPolicyEnvKey)
}

func GetServiceControllerImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetServiceControllerImageName(),
		PullPolicy: GetServiceControllerImagePullPolicy(),
	}
}

func GetControllerPodmanImageName() string {
	image := os.Getenv(ControllerPodmanImageEnvKey)
	if image == "" {
		imageRegistry := GetImageRegistry()
		return strings.Join([]string{imageRegistry, ControllerPodmanImageName}, "/")
	} else {
		return image
	}
}

func GetControllerPodmanImagePullPolicy() string {
	return getPullPolicy(ControllerPodmanPullPolicyEnvKey)
}

func GetControllerPodmanImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetControllerPodmanImageName(),
		PullPolicy: GetControllerPodmanImagePullPolicy(),
	}
}

func GetConfigSyncImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetConfigSyncImageName(),
		PullPolicy: GetConfigSyncImagePullPolicy(),
	}
}

func GetConfigSyncImageName() string {
	image := os.Getenv(ConfigSyncImageEnvKey)
	if image == "" {
		imageRegistry := GetImageRegistry()
		return strings.Join([]string{imageRegistry, ConfigSyncImageName}, "/")
	} else {
		return image
	}
}

func GetConfigSyncImagePullPolicy() string {
	return getPullPolicy(ConfigSyncPullPolicyEnvKey)
}

func GetFlowCollectorImageName() string {
	image := os.Getenv(FlowCollectorImageEnvKey)
	if image == "" {
		imageRegistry := GetImageRegistry()
		return strings.Join([]string{imageRegistry, FlowCollectorImageName}, "/")
	} else {
		return image
	}
}

func GetFlowCollectorImagePullPolicy() string {
	return getPullPolicy(FlowCollectorPullPolicyEnvKey)
}

func GetFlowCollectorImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetFlowCollectorImageName(),
		PullPolicy: GetFlowCollectorImagePullPolicy(),
	}
}

func GetPrometheusServerImageName() string {
	image := os.Getenv(PrometheusServerImageEnvKey)
	if image == "" {
		imageRegistry := GetPrometheusImageRegistry()
		return strings.Join([]string{imageRegistry, PrometheusServerImageName}, "/")
	} else {
		return image
	}
}

func GetPrometheusServerImagePullPolicy() string {
	return getPullPolicy(PrometheusServerPullPolicyEnvKey)
}

func GetPrometheusServerImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetPrometheusServerImageName(),
		PullPolicy: GetPrometheusServerImagePullPolicy(),
	}
}

func GetImageRegistry() string {
	imageRegistry := os.Getenv(SkupperImageRegistryEnvKey)
	if imageRegistry == "" {
		return DefaultImageRegistry
	}
	return imageRegistry
}

func GetPrometheusImageRegistry() string {
	imageRegistry := os.Getenv(PrometheusImageRegistryEnvKey)
	if imageRegistry == "" {
		return PrometheusImageRegistry
	}
	return imageRegistry
}

func GetSiteControllerImageName() string {
	imageRegistry := GetImageRegistry()
	return strings.Join([]string{imageRegistry, SiteControllerImageName}, "/")
}

func GetSha(imageName string) string {
	// Pull the image
	pullCmd := exec.Command("docker", "pull", imageName)
	if err := pullCmd.Run(); err != nil {
		fmt.Printf("Error pulling image: %v", err)
		return err.Error()
	}

	// Get the image digest
	digestCmd := exec.Command("docker", "inspect", "--format='{{index .RepoDigests 0}}'", imageName)
	digestBytes, err := digestCmd.Output()
	if err != nil {
		fmt.Printf("Error getting image digest: %v", err)
		return err.Error()
	}

	imageWithoutTag := strings.Split(imageName, ":")[0]

	// Extract and print the digest
	parsedDigest := strings.ReplaceAll(strings.ReplaceAll(string(digestBytes), "'", ""), "\n", "")
	digest := strings.TrimPrefix(strings.Trim(parsedDigest, "'"), imageWithoutTag+"@")

	return digest
}

func GetOauthProxyImageName() string {
	image := os.Getenv(OauthProxyImageEnvKey)
	if image == "" {
		imageRegistry := GetOauthProxyImageRegistry()
		return strings.Join([]string{imageRegistry, OauthProxyImageName}, "/")
	} else {
		return image
	}
}

func GetOauthProxyImagePullPolicy() string {
	return getPullPolicy(OauthProxyPullPolicyEnvKey)
}

func GetOauthProxyImageDetails() types.ImageDetails {
	return types.ImageDetails{
		Name:       GetOauthProxyImageName(),
		PullPolicy: GetOauthProxyImagePullPolicy(),
	}
}

func GetOauthProxyImageRegistry() string {
	imageRegistry := os.Getenv(OauthProxyRegistryEnvKey)
	if imageRegistry == "" {
		return OauthProxyImageRegistry
	}
	return imageRegistry
}
