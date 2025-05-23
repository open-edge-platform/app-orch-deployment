package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
)

func (cli *VanClient) SiteConfigRemove(ctx context.Context) error {
	return cli.KubeClient.CoreV1().ConfigMaps(cli.Namespace).Delete(ctx, types.SiteConfigMapName, metav1.DeleteOptions{})
}
