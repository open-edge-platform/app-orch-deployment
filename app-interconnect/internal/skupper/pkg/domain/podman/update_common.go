package podman

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	clientpodman "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/podman"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/domain"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/images"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/version"
)

func NewVersionUpdateTask(cli *clientpodman.PodmanRestClient) *VersionUpdateTask {
	return &VersionUpdateTask{
		cli:     cli,
		version: version.Version,
	}
}

type VersionUpdateTask struct {
	cli     *clientpodman.PodmanRestClient
	version string
}

func (v *VersionUpdateTask) Info() string {
	return "Updates current version number"
}

func (v *VersionUpdateTask) AppliesTo(siteVersion string) bool {
	curVersion := utils.ParseVersion(siteVersion)
	return !(&curVersion).IsUndefined() && utils.LessRecentThanVersion(siteVersion, v.version)
}

func (v *VersionUpdateTask) Version() string {
	return "*"
}

func (v *VersionUpdateTask) Priority() domain.UpdatePriority {
	return domain.PriorityLast
}

func (v *VersionUpdateTask) Run(context.Context) *domain.UpdateResult {
	var res = &domain.UpdateResult{}
	ch := NewRouterConfigHandlerPodman(v.cli)
	cfg, err := ch.GetRouterConfig()
	if err != nil {
		res.AddErrors(fmt.Errorf("error retrieving router config: %s", err))
		return res
	}
	siteMeta := cfg.GetSiteMetadata()
	siteMeta.Version = v.version
	cfg.SetSiteMetadata(&siteMeta)
	if err = ch.SaveRouterConfig(cfg); err != nil {
		res.AddErrors(fmt.Errorf("error saving router config: %s", err))
		return res
	}
	res.AddChange(fmt.Sprintf("updated site version to %s", v.version))
	return res
}

func NewContainerImagesTask(cli *clientpodman.PodmanRestClient) *ContainerImagesTask {
	return &ContainerImagesTask{
		cli:     cli,
		version: version.Version,
	}
}

type ContainerImagesTask struct {
	cli     *clientpodman.PodmanRestClient
	version string
}

func (u *ContainerImagesTask) Info() string {
	return "Updates skupper podman container images"
}

func (u *ContainerImagesTask) AppliesTo(siteVersion string) bool {
	curVersion := utils.ParseVersion(siteVersion)
	return !(&curVersion).IsUndefined() && utils.LessRecentThanVersion(siteVersion, u.version)
}

func (u *ContainerImagesTask) Version() string {
	return "*"
}

func (u *ContainerImagesTask) Priority() domain.UpdatePriority {
	return domain.PriorityNormal
}

func (u *ContainerImagesTask) Run(ctx context.Context) *domain.UpdateResult {
	var result = &domain.UpdateResult{}
	sh := NewSitePodmanHandlerFromCli(u.cli)
	site, err := sh.Get()
	if err != nil {
		result.AddErrors(fmt.Errorf("error retrieving site info: %s", err))
		return result
	}
	// updating images for all running components
	for _, dep := range site.GetDeployments() {
		for _, cmp := range dep.GetComponents() {
			var image string
			switch cmp.Name() {
			case types.TransportDeploymentName:
				image = images.GetRouterImageName()
			case types.FlowCollectorContainerName:
				image = images.GetFlowCollectorImageName()
			case types.ControllerPodmanContainerName:
				image = images.GetControllerPodmanImageName()
			case types.PrometheusDeploymentName:
				image = images.GetPrometheusServerImageName()
			}
			if image != cmp.GetImage() {
				_, err = u.cli.ContainerUpdateImage(ctx, cmp.Name(), image)
				if err != nil {
					result.AddErrors(fmt.Errorf("error updating container: %s - image: %s - %s",
						cmp.Name(), image, err))
					return result
				}
				result.AddChange(fmt.Sprintf("container updated: %s - image: %s", cmp.Name(), image))
			}
		}
	}
	// updating service containers
	updSvcResult := u.updateServiceContainers(ctx)
	if len(updSvcResult.Errors) > 0 {
		result.AddErrors(updSvcResult.Errors...)
		return result
	}
	if updSvcResult.Changed() {
		result.AddChange(updSvcResult.GetChanges()...)
	}
	return result
}

func (u *ContainerImagesTask) updateServiceContainers(ctx context.Context) domain.UpdateResult {
	var result domain.UpdateResult
	sh := NewServiceHandlerPodman(u.cli)
	services, err := sh.List()
	if err != nil {
		result.AddErrors(fmt.Errorf("error listing services: %s", err))
		return result
	}
	for _, svc := range services {
		svcPodman := svc.(*Service)
		c, err := u.cli.ContainerInspect(svcPodman.ContainerName)
		if err != nil {
			result.AddErrors(fmt.Errorf("error retrieving container info for %s: %s",
				svcPodman.ContainerName, err))
			return result
		}
		if c.Image != images.GetRouterImageName() {
			_, err = u.cli.ContainerUpdateImage(ctx, svcPodman.ContainerName, images.GetRouterImageName())
			if err != nil {
				result.AddErrors(fmt.Errorf("error updating service container image for %s: %s",
					svcPodman.ContainerName, err))
				return result
			}
			result.AddChange(fmt.Sprintf("service container updated: %s - image: %s",
				svcPodman.ContainerName, images.GetRouterImageName()))
		}
	}
	return result
}
