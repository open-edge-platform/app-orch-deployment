package podman

import (
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/utils"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
)

const (
	SharedTlsCertificates = "skupper-router-certs"
)

var (
	Username                = utils.ReadUsername()
	SkupperContainerVolumes = []string{
		"skupper-services",
		"skupper-local-server",
		"skupper-internal",
		"skupper-site-server",
		SharedTlsCertificates,
		types.ConsoleServerSecret,
		types.ConsoleUsersSecret,
		types.NetworkStatusConfigMapName,
		"prometheus-server-config",
		"prometheus-storage-volume",
	}
)

func OwnedBySkupper(resource string, labels map[string]string) error {
	notOwnedErr := fmt.Errorf("%s is not owned by Skupper", resource)
	if labels == nil {
		return notOwnedErr
	}
	if app, ok := labels["application"]; !ok || app != types.AppName {
		return notOwnedErr
	}
	return nil
}
