// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	corev1 "k8s.io/api/core/v1"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ValidUID         = "123456-123456-123456"
	ValidProjectID   = "0000-1111-2222-3333-4444"
	ValidClusterName = "cluster-01234567"
)

type FakeDeploymentV1 struct {
	mock.Mock
}

func (c *FakeDeploymentV1) Deployments(namespace string) v1beta1.DeploymentInterface {
	return &FakeDeployments{c, namespace}
}

func (c *FakeDeploymentV1) DeploymentClusters(namespace string) v1beta1.DeploymentClusterInterface {
	return &FakeDeploymentClusters{c, namespace}
}

func (c *FakeDeploymentV1) Clusters(namespace string) v1beta1.ClusterInterface {
	return &FakeClusters{c, namespace}
}

func (c *FakeDeploymentV1) APIExtensions(namespace string) v1beta1.APIExtensionInterface {
	return &FakeAPIExtensions{c, namespace}
}

func (c *FakeDeploymentV1) GrafanaExtensions(namespace string) v1beta1.GrafanaExtensionInterface {
	return &FakeGrafanaExtensions{c, namespace}
}

type FakeDeployments struct {
	Fake *FakeDeploymentV1
	ns   string
}

type FakeDeploymentClusters struct {
	Fake *FakeDeploymentV1
	ns   string
}

type FakeClusters struct {
	Fake *FakeDeploymentV1
	ns   string
}

type FakeAPIExtensions struct {
	Fake *FakeDeploymentV1
	ns   string
}

type FakeGrafanaExtensions struct {
	Fake *FakeDeploymentV1
	ns   string
}

func (c *FakeDeployments) List(ctx context.Context, opts metav1.ListOptions) (*deploymentv1beta1.DeploymentList, error) {
	args := c.Fake.Called(ctx, opts)
	return args.Get(0).(*deploymentv1beta1.DeploymentList), args.Error(1)
}

func (c *FakeDeployments) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentv1beta1.Deployment, error) {
	args := c.Fake.Called(ctx, name, opts)
	return args.Get(0).(*deploymentv1beta1.Deployment), args.Error(1)
}

func (c *FakeDeployments) Create(ctx context.Context, deployment *deploymentv1beta1.Deployment, opts metav1.CreateOptions) (*deploymentv1beta1.Deployment, error) {
	// set constant values since func CreateDeployment will generate random values
	deployment.Name = "test-deployment"
	deployment.Namespace = ValidProjectID
	deployment.Labels["app.kubernetes.io/instance"] = "test-deployment"
	deployment.Labels[string(deploymentv1beta1.AppOrchActiveProjectID)] = ValidProjectID
	deployment.Labels[string(deploymentv1beta1.ClusterName)] = ValidClusterName
	deployment.ObjectMeta.UID = ValidUID

	if len(deployment.Spec.Applications) > 0 {
		deployment.Spec.Applications[0].ProfileSecretName = ""
		deployment.Spec.Applications[0].Namespace = "test-deployment"
		deployment.Spec.Applications[0].ValueSecretName = ""
		deployment.Spec.Applications[0].HelmApp.RepoSecretName = ""
		deployment.Spec.Applications[0].HelmApp.ImageRegistrySecretName = ""
		deployment.Spec.Applications[0].EnableServiceExport = true
	}

	deployment.Spec.NetworkRef = corev1.ObjectReference{
		Name:       "network-1",
		Kind:       "Network",
		APIVersion: "network.edge-orchestrator.intel/v1",
	}

	args := c.Fake.Called(ctx, deployment, opts)
	return args.Get(0).(*deploymentv1beta1.Deployment), args.Error(1)
}

func (c *FakeDeployments) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	args := c.Fake.Called(ctx, name, opts)
	return args.Error(0)
}

func (c *FakeDeployments) Update(ctx context.Context, name string, deployment *deploymentv1beta1.Deployment, opts metav1.UpdateOptions) (*deploymentv1beta1.Deployment, error) {
	deployment.ObjectMeta.UID = ValidUID
	deployment.Namespace = ValidProjectID
	deployment.Labels[string(deploymentv1beta1.AppOrchActiveProjectID)] = ValidProjectID
	deployment.Labels[string(deploymentv1beta1.ClusterName)] = ValidClusterName

	if len(deployment.Spec.Applications) > 0 {
		deployment.Spec.Applications[0].ProfileSecretName = ""
		deployment.Spec.Applications[0].ValueSecretName = ""
		deployment.Spec.Applications[0].HelmApp.RepoSecretName = ""
		deployment.Spec.Applications[0].HelmApp.ImageRegistrySecretName = ""
	}

	args := c.Fake.Called(ctx, name, deployment, opts)
	return args.Get(0).(*deploymentv1beta1.Deployment), args.Error(1)
}

func (c *FakeDeploymentClusters) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentv1beta1.DeploymentCluster, error) {
	args := c.Fake.Called(ctx, name, opts)
	return args.Get(0).(*deploymentv1beta1.DeploymentCluster), args.Error(1)
}

func (c *FakeDeploymentClusters) List(ctx context.Context, opts metav1.ListOptions) (*deploymentv1beta1.DeploymentClusterList, error) {
	args := c.Fake.Called(ctx, opts)
	return args.Get(0).(*deploymentv1beta1.DeploymentClusterList), args.Error(1)
}

func (c *FakeClusters) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentv1beta1.Cluster, error) {
	args := c.Fake.Called(ctx, name, opts)
	return args.Get(0).(*deploymentv1beta1.Cluster), args.Error(1)
}

func (c *FakeClusters) List(ctx context.Context, opts metav1.ListOptions) (*deploymentv1beta1.ClusterList, error) {
	args := c.Fake.Called(ctx, opts)
	return args.Get(0).(*deploymentv1beta1.ClusterList), args.Error(1)
}

func (c *FakeAPIExtensions) List(ctx context.Context, opts metav1.ListOptions) (*deploymentv1beta1.APIExtensionList, error) {
	args := c.Fake.Called(ctx, opts)
	return args.Get(0).(*deploymentv1beta1.APIExtensionList), args.Error(1)
}

func (c *FakeAPIExtensions) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentv1beta1.APIExtension, error) {
	args := c.Fake.Called(ctx, name, opts)
	return args.Get(0).(*deploymentv1beta1.APIExtension), args.Error(1)
}

func (c *FakeAPIExtensions) Create(ctx context.Context, apiExtension *deploymentv1beta1.APIExtension, opts metav1.CreateOptions) (*deploymentv1beta1.APIExtension, error) {
	apiExtension.Name = "test-deployment"
	apiExtension.Spec.Project = "test-project"
	apiExtension.Status.TokenSecretRef.Name = "test-token"
	apiExtension.Status.TokenSecretRef.GeneratedToken = "this-is-a-test-token"
	args := c.Fake.Called(ctx, apiExtension, opts)
	return args.Get(0).(*deploymentv1beta1.APIExtension), args.Error(1)
}

func (c *FakeAPIExtensions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	args := c.Fake.Called(ctx, name, opts)
	return args.Error(0)
}

func (c *FakeGrafanaExtensions) Create(ctx context.Context, grafanaExtension *deploymentv1beta1.GrafanaExtension, opts metav1.CreateOptions) (*deploymentv1beta1.GrafanaExtension, error) {
	grafanaExtension.Name = "test-deployment"
	grafanaExtension.Spec.Project = "test-project"
	grafanaExtension.Spec.ArtifactRef.Artifact = "test-artifact"

	args := c.Fake.Called(ctx, grafanaExtension, opts)
	return args.Get(0).(*deploymentv1beta1.GrafanaExtension), args.Error(1)
}

func (c *FakeGrafanaExtensions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	args := c.Fake.Called(ctx, name, opts)
	return args.Error(0)
}

func (c *FakeGrafanaExtensions) Get(ctx context.Context, name string, opts metav1.GetOptions) (*deploymentv1beta1.GrafanaExtension, error) {
	args := c.Fake.Called(ctx, name, opts)
	return args.Get(0).(*deploymentv1beta1.GrafanaExtension), args.Error(1)
}

func (c *FakeGrafanaExtensions) List(ctx context.Context, opts metav1.ListOptions) (*deploymentv1beta1.GrafanaExtensionList, error) {
	args := c.Fake.Called(ctx, opts)
	return args.Get(0).(*deploymentv1beta1.GrafanaExtensionList), args.Error(1)
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) AuthCatalog(ctx context.Context, name string, version string) error {
	args := m.Called(ctx, name, version)
	return args.Error(0)
}

var AnyContextTodo = mock.AnythingOfType("*context.todoCtx")
var AnyContextEmpty = mock.AnythingOfType("*context.emptyCtx")
var AnyContext = mock.AnythingOfType("*context.timerCtx")
var AnyContextValue = mock.AnythingOfType("*context.valueCtx")

const (
	dpname     = "wordpress"
	dpnamebad  = "doesnotexist"
	dpver      = "0.1.0"
	dpprof     = "default"
	appname    = "wordpress"
	appver     = "0.1.0"
	appprof    = "default"
	appvalues  = `{"foo": "bar"}`
	chartname  = "wordpress"
	chartver   = "15.2.42"
	helmreg    = "helmregistry"
	helmurl    = "https://charts.bitnami.com/bitnami"
	helmuser   = "helmuser"
	helmpass   = "helmpass"
	helmcacert = "helmcacert-goes-here"
	dockerreg  = "dockerregistry"
	dockerurl  = "https://hub.docker.com/"
	dockeruser = "dockeruser"
	dockerpass = "dockerpass"
)

var (
	appRef catalog.ApplicationReference
)

var AnyGetDpReq = &catalog.GetDeploymentPackageRequest{
	DeploymentPackageName: dpname,
	Version:               dpver,
}

var DpRespNsName = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:                  dpname,
		Version:               dpver,
		ApplicationReferences: []*catalog.ApplicationReference{},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		Namespaces: []*catalog.Namespace{
			{
				Name:        "ns-test",
				Labels:      map[string]string{"test-label-key": "test-label-value"},
				Annotations: map[string]string{"test-ann-key": "test-ann-value"},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{},
		Artifacts:  []*catalog.ArtifactReference{},
		DefaultNamespaces: map[string]string{
			appname: "test-deployment",
		},
	},
}

var DpRespGood = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:    dpname,
		Version: dpver,
		ApplicationReferences: []*catalog.ApplicationReference{
			&appRef,
		},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{},
		Artifacts:  []*catalog.ArtifactReference{},
		DefaultNamespaces: map[string]string{
			appname: "test-deployment",
		},
	},
}

var DpAPIRespGood = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:    dpname,
		Version: dpver,
		ApplicationReferences: []*catalog.ApplicationReference{
			&appRef,
		},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{
			{
				Name:        "test-name",
				DisplayName: "test-displayname",
				Description: "test-description",
				Version:     "test-version",
				Endpoints: []*catalog.Endpoint{
					{
						ServiceName:  "test-servicename",
						ExternalPath: "test-externalpath",
						InternalPath: "test-internalpath",
						Scheme:       "test-scheme",
						AuthType:     "test-authtype",
						AppName:      "test-appname",
					},
				},
				UiExtension: &catalog.UIExtension{
					Label:       "test-label",
					ServiceName: "test-servicename",
					Description: "test-description",
					FileName:    "test-filename",
					AppName:     "test-appname",
					ModuleName:  "test-modulename",
				},
			},
		},
		Artifacts: []*catalog.ArtifactReference{},
		DefaultNamespaces: map[string]string{
			appname: "test-deployment",
		},
	},
}

var DpNoAPIRespGood = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:    dpname,
		Version: dpver,
		ApplicationReferences: []*catalog.ApplicationReference{
			&appRef,
		},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{},
		Artifacts:  []*catalog.ArtifactReference{},
		DefaultNamespaces: map[string]string{
			appname: "test-deployment",
		},
	},
}

var DpGrafanaRespGood = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:    dpname,
		Version: dpver,
		ApplicationReferences: []*catalog.ApplicationReference{
			&appRef,
		},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{},
		Artifacts: []*catalog.ArtifactReference{
			{
				Name:    "sample-dashboard",
				Purpose: "grafana",
			},
		},
		DefaultNamespaces: map[string]string{},
	},
}

var DpNoGrafanaRespGood = catalog.GetDeploymentPackageResponse{
	DeploymentPackage: &catalog.DeploymentPackage{
		Name:    dpname,
		Version: dpver,
		ApplicationReferences: []*catalog.ApplicationReference{
			&appRef,
		},
		Profiles: []*catalog.DeploymentProfile{
			{
				Name: dpprof,
				ApplicationProfiles: map[string]string{
					appname: appprof,
				},
			},
		},
		DefaultProfileName: dpprof,
		ApplicationDependencies: []*catalog.ApplicationDependency{
			{
				Name:     appname,
				Requires: "dependency",
			},
		},
		Extensions: []*catalog.APIExtension{},
		Artifacts: []*catalog.ArtifactReference{
			{},
		},
		DefaultNamespaces: map[string]string{},
	},
}

var AnyGetAppReq = &catalog.GetApplicationRequest{}

var AppResp = catalog.GetApplicationResponse{
	Application: &catalog.Application{
		Name:             appname,
		Version:          appver,
		ChartName:        chartname,
		ChartVersion:     chartver,
		HelmRegistryName: helmreg,
		Profiles: []*catalog.Profile{
			{
				Name:        appprof,
				ChartValues: appvalues,
			},
		},
		DefaultProfileName: appprof,
		ImageRegistryName:  dockerreg,
	},
}

var AppHelmResp = catalog.GetApplicationResponse{
	Application: &catalog.Application{
		Name:             appname,
		Version:          appver,
		ChartName:        chartname,
		ChartVersion:     chartver,
		HelmRegistryName: helmreg,
		Profiles: []*catalog.Profile{
			{
				Name:        appprof,
				ChartValues: appvalues,
			},
		},
		DefaultProfileName: appprof,
		ImageRegistryName:  helmreg,
	},
}

var parameterTemplate1 = &catalog.ParameterTemplate{
	Name:        "global.admin_password",
	DisplayName: "test-tp-value",
	Default:     "",
	Secret:      true,
	Mandatory:   true,
}

var parameterTemplate2 = &catalog.ParameterTemplate{
	Name:        "globals.another_password",
	DisplayName: "test-tp2-value",
	Default:     "",
	Secret:      true,
	Mandatory:   true,
}

var AppHelmPtResp = catalog.GetApplicationResponse{
	Application: &catalog.Application{
		Name:             appname,
		Version:          appver,
		ChartName:        chartname,
		ChartVersion:     chartver,
		HelmRegistryName: helmreg,
		Profiles: []*catalog.Profile{
			{
				Name:        appprof,
				ChartValues: appvalues,
				ParameterTemplates: []*catalog.ParameterTemplate{
					parameterTemplate1,
				},
			},
		},
		DefaultProfileName: appprof,
		ImageRegistryName:  helmreg,
	},
}

var AppHelmPt2Resp = catalog.GetApplicationResponse{
	Application: &catalog.Application{
		Name:             appname,
		Version:          appver,
		ChartName:        chartname,
		ChartVersion:     chartver,
		HelmRegistryName: helmreg,
		Profiles: []*catalog.Profile{
			{
				Name:        appprof,
				ChartValues: appvalues,
				ParameterTemplates: []*catalog.ParameterTemplate{
					parameterTemplate1,
					parameterTemplate2,
				},
			},
		},
		DefaultProfileName: appprof,
		ImageRegistryName:  helmreg,
	},
}

var AnyGetRegReq = &catalog.GetRegistryRequest{
	RegistryName:      helmreg,
	ShowSensitiveInfo: true,
}

var HelmRegResp = catalog.GetRegistryResponse{
	Registry: &catalog.Registry{
		Name:      helmreg,
		RootUrl:   helmurl,
		Username:  helmuser,
		AuthToken: helmpass,
		Type:      "HELM",
		Cacerts:   helmcacert,
	},
}

var AnyGetDockerRegReq = &catalog.GetRegistryRequest{
	RegistryName:      dockerreg,
	ShowSensitiveInfo: true,
}

var DockerRegResp = catalog.GetRegistryResponse{
	Registry: &catalog.Registry{
		Name:      dockerreg,
		RootUrl:   dockerurl,
		Username:  dockeruser,
		AuthToken: dockerpass,
		Type:      "IMAGE",
	},
}

var AnyArtReq = &catalog.ListArtifactsRequest{}

var ArtiResp = catalog.ListArtifactsResponse{
	Artifacts: []*catalog.Artifact{
		{
			Name:     "sample-dashboard",
			MimeType: "application/json",
			Artifact: []byte("{\"test\":\"foo\"}"),
		},
	},
}
