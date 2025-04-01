package podman

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/api/types"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/container"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client/podman"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/domain"
)

type SkupperDeployment struct {
	*domain.SkupperDeploymentCommon
	Name           string
	Aliases        []string
	VolumeMounts   map[string]string
	Networks       []string
	SELinuxDisable bool
}

func (s *SkupperDeployment) GetName() string {
	return s.Name
}

type SkupperDeploymentHandler struct {
	cli *podman.PodmanRestClient
}

func NewSkupperDeploymentHandlerPodman(cli *podman.PodmanRestClient) *SkupperDeploymentHandler {
	return &SkupperDeploymentHandler{
		cli: cli,
	}
}

// Deploy deploys each component as a container
func (s *SkupperDeploymentHandler) Deploy(ctx context.Context, deployment domain.SkupperDeployment) error {
	var err error
	var cleanupContainers []string

	defer func() {
		if err != nil {
			for _, containerName := range cleanupContainers {
				_ = s.cli.ContainerStop(containerName)
				_ = s.cli.ContainerRemove(containerName)
			}
		}
	}()

	if len(deployment.GetComponents()) > 1 {
		return fmt.Errorf("podman implementation currently allows only one component per deployment")
	}

	podmanDeployment := deployment.(*SkupperDeployment)
	for _, component := range deployment.GetComponents() {

		// Pulling image first
		err = s.cli.ImagePull(ctx, component.GetImage())
		if err != nil {
			return err
		}

		// Setting network aliases
		networkMap := map[string]container.ContainerNetworkInfo{}
		for _, network := range podmanDeployment.Networks {
			networkMap[network] = container.ContainerNetworkInfo{
				Aliases: podmanDeployment.Aliases,
			}
		}

		// Defining the mounted volumes
		mounts := []container.Volume{}
		fileMounts := []container.FileMount{}
		for volumeName, destDir := range podmanDeployment.VolumeMounts {
			fileOrDir := strings.HasPrefix(volumeName, "/")
			if fileOrDir {
				var mount *container.FileMount
				mount = &container.FileMount{
					Source:      volumeName,
					Destination: destDir,
					Options:     []string{"z"},
				}
				fileMounts = append(fileMounts, *mount)
			} else {
				var volume *container.Volume
				volume, err = s.cli.VolumeInspect(volumeName)
				if err != nil {
					err = fmt.Errorf("error reading volume %s - %v", volumeName, err)
					return err
				}
				volume.Destination = destDir
				volume.Mode = "z" // shared between containers
				mounts = append(mounts, *volume)
			}
		}

		// Ports
		ports := []container.Port{}
		for _, siteIngress := range component.GetSiteIngresses() {
			ports = append(ports, container.Port{
				Host:     strconv.Itoa(siteIngress.GetPort()),
				HostIP:   siteIngress.GetHost(),
				Target:   strconv.Itoa(siteIngress.GetTarget().GetPort()),
				Protocol: "tcp",
			})
		}

		// Defining the container
		labels := component.GetLabels()
		labels[types.ComponentAnnotation] = deployment.GetName()
		c := &container.Container{
			Name:           component.Name(),
			Image:          component.GetImage(),
			Env:            component.GetEnv(),
			Labels:         labels,
			Networks:       networkMap,
			Mounts:         mounts,
			FileMounts:     fileMounts,
			Ports:          ports,
			RestartPolicy:  "always",
			MaxMemoryBytes: component.GetMemoryLimit(),
			MaxCpus:        component.GetCpus(),
		}

		if podmanDeployment.SELinuxDisable {
			if c.Annotations == nil {
				c.Annotations = map[string]string{}
			}
			c.Annotations["io.podman.annotations.label"] = "disable"
		}

		err = s.cli.ContainerCreate(c)
		if err != nil {
			return fmt.Errorf("error creating skupper component: %s - %v", c.Name, err)
		}
		cleanupContainers = append(cleanupContainers, c.Name)

		err = s.cli.ContainerStart(c.Name)
		if err != nil {
			return fmt.Errorf("error starting skupper component: %s - %v", c.Name, err)
		}
	}

	return nil
}

func (s *SkupperDeploymentHandler) Undeploy(name string) error {
	containers, err := s.cli.ContainerList()
	if err != nil {
		return fmt.Errorf("error listing containers - %w", err)
	}

	stopContainers := []string{}
	for _, c := range containers {
		if component, ok := c.Labels[types.ComponentAnnotation]; ok && component == name {
			stopContainers = append(stopContainers, c.Name)
		}
	}

	if len(stopContainers) == 0 {
		return nil
	}

	for _, c := range stopContainers {
		_ = s.cli.ContainerStop(c)
		_ = s.cli.ContainerRemove(c)
	}
	return nil
}

func (s *SkupperDeploymentHandler) List() ([]domain.SkupperDeployment, error) {
	depMap := map[string]domain.SkupperDeployment{}

	list, err := s.cli.ContainerList()
	if err != nil {
		return nil, fmt.Errorf("error retrieving container list - %w", err)
	}

	var depList []domain.SkupperDeployment

	componentHandler := NewSkupperComponentHandlerPodman(s.cli)
	components, err := componentHandler.List()
	if err != nil {
		return nil, fmt.Errorf("error retrieving existing skupper components - %w", err)
	}

	for _, c := range list {
		ci, err := s.cli.ContainerInspect(c.Name)
		if err != nil {
			return nil, fmt.Errorf("error retrieving container information for %s - %w", c.Name, err)
		}
		if ci.Labels == nil {
			continue
		}
		deployName, ok := ci.Labels[types.ComponentAnnotation]
		if !ok {
			continue
		}
		var aliases []string
		for _, aliases = range ci.NetworkAliases() {
			break
		}
		mounts := map[string]string{}
		for _, mount := range ci.Mounts {
			mounts[mount.Name] = mount.Destination
		}
		deployment := &SkupperDeployment{
			SkupperDeploymentCommon: &domain.SkupperDeploymentCommon{},
			Name:                    deployName,
			Aliases:                 aliases,
			VolumeMounts:            mounts,
			Networks:                ci.NetworkNames(),
		}
		depMap[deployName] = deployment

		depComponents := []domain.SkupperComponent{}
		for _, component := range components {
			if compOwner, ok := component.GetLabels()[types.ComponentAnnotation]; ok && compOwner == deployName {
				depComponents = append(depComponents, component)
			}
		}
		deployment.Components = depComponents
		depList = append(depList, deployment)
	}

	return depList, nil
}
