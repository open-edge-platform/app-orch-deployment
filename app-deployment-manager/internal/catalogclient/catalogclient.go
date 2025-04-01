// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package catalogclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"google.golang.org/grpc/codes"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/open-edge-platform/orch-library/go/dazl"
)

var log = dazl.GetPackageLogger()

const (
	ArtifactGrafanaPurpose string = "grafana"
)

// Reference: https://fleet.rancher.io/gitrepo-add#using-private-helm-repositories
type HelmCredential struct {
	Username string
	Password string
	CaCerts  string
}

type DockerCredential struct {
	Username string
	Password string
}

type IgnoreResource struct {
	Name      string
	Kind      string
	Namespace string
}

type HelmApp struct {
	Name                       string
	Repo                       string
	ImageRegistry              string
	Chart                      string
	Version                    string
	Profile                    string
	Values                     string
	DependsOn                  []string
	RedeployAfterUpdate        bool
	DefaultNamespace           string
	HelmCredential             HelmCredential
	DockerCredential           DockerCredential
	IgnoreResource             []IgnoreResource
	RequiredDeploymentPackages []RequiredDeploymentPackage
	ParameterTemplates         []ParameterTemplate
}

type ParameterTemplate struct {
	Name        string
	DisplayName string
	Type        string
	Mandatory   bool
	Secret      bool
}

type RequiredDeploymentPackage struct {
	Name    string
	Version string
	Profile string
}

func (r *RequiredDeploymentPackage) GetID() string {
	return GetDeploymentPackageID(r.Name, r.Version, r.Profile)
}

// CatalogClient interface is used to query the Catalog Service.
// Currently only a subset of the full Catalog Service API is used.
//
//go:generate mockery --name CatalogClient --filename mockery_catalogclient.go --structname MockeryCatalogClient --output mockery
type CatalogClient interface {
	GetDeploymentPackage(ctx context.Context, in *catalog.GetDeploymentPackageRequest, opts ...grpc.CallOption) (*catalog.GetDeploymentPackageResponse, error)
	UpdateDeploymentPackage(ctx context.Context, in *catalog.UpdateDeploymentPackageRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetApplication(ctx context.Context, in *catalog.GetApplicationRequest, opts ...grpc.CallOption) (*catalog.GetApplicationResponse, error)
	GetRegistry(ctx context.Context, in *catalog.GetRegistryRequest, opts ...grpc.CallOption) (*catalog.GetRegistryResponse, error)
	ListArtifacts(ctx context.Context, in *catalog.ListArtifactsRequest, opts ...grpc.CallOption) (*catalog.ListArtifactsResponse, error)
	GetArtifact(ctx context.Context, in *catalog.GetArtifactRequest, opts ...grpc.CallOption) (*catalog.GetArtifactResponse, error)
}

// Return a HelmApp populated from the details of a catalog.Application and catalog.Registry
func newHelmApp(app *catalog.Application, helmregistry *catalog.Registry, imageregistry *catalog.Registry, values string, profile string) HelmApp {
	redeploy := os.Getenv("REDEPLOY_AFTER_UPDATE") == "true"

	ha := HelmApp{
		Name:                app.Name,
		Chart:               app.ChartName,
		Version:             app.ChartVersion,
		Repo:                helmregistry.RootUrl,
		Profile:             profile,
		Values:              values,
		ParameterTemplates:  []ParameterTemplate{},
		RedeployAfterUpdate: redeploy,
		HelmCredential: HelmCredential{
			Username: helmregistry.Username,
			Password: helmregistry.AuthToken,
			CaCerts:  helmregistry.Cacerts,
		},
	}
	if imageregistry != nil {
		ha.ImageRegistry = imageregistry.RootUrl
		ha.DockerCredential.Username = imageregistry.Username
		ha.DockerCredential.Password = imageregistry.AuthToken
	}
	for _, ign := range app.IgnoredResources {
		ha.IgnoreResource = append(ha.IgnoreResource, IgnoreResource{
			Name:      ign.Name,
			Kind:      ign.Kind,
			Namespace: ign.Namespace,
		})
	}
	return ha
}

var NewCatalogClient = func() (CatalogClient, error) {
	addr, ok := os.LookupEnv("CATALOG_SERVICE_ADDRESS")
	if !ok {
		return nil, fmt.Errorf("catalog service address is not set")
	}

	// nolint:staticcheck
	conn, err := grpc.DialContext(context.Background(), addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))
	if err != nil {
		return nil, err
	}

	return catalog.NewCatalogServiceClient(conn), nil
}

func GetDeploymentPackageID(name, version, profile string) string {
	return fmt.Sprintf("%s/%s/%s", name, version, profile)
}

func getAppValues(app *catalog.Application, profmap map[string]string) (string, bool) {
	profile, ok := profmap[app.Name]
	if ok {
		for _, p := range app.Profiles {
			if p.Name == profile {
				return p.ChartValues, true
			}
		}
	}
	return "", false
}

func getProfileMap(cps []*catalog.DeploymentProfile, name string) map[string]string {
	for _, cp := range cps {
		if cp.Name == name {
			return cp.ApplicationProfiles
		}
	}
	return map[string]string{}
}

// Allow overrriding the Connect and CatalogLookup functions for testing error handling in the controller.
// The test framework can easily return errors to the controller's reconcile loop
// to test that the controller reports the error correctly and recovers when the error resolves.

// Connect wraps setting up a gRPC connection to the Catalog Service.
// nolint:staticcheck
var Connect = func(ctx context.Context, serverAddr string) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return conn, err
}

// CatalogLookup looks up a Deployment Package in the Catalog Service. It fetches the Applications, Registries,
// and Profile corresponding to the Deployment Package, and reformat into a slice of HelmApp for ease of use.
var CatalogLookupDPAndHelmApps = func(ctx context.Context, client CatalogClient, dpName string, dpVersion string, profileName string) (*catalog.DeploymentPackage, *[]HelmApp, string, error) {
	var helmapps *[]HelmApp
	var err error

	log.Info(fmt.Sprintf("Look up Deployment Package Name: %s | Version %s", dpName, dpVersion))
	request := catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: dpName,
		Version:               dpVersion,
	}

	response, err := client.GetDeploymentPackage(ctx, &request)
	if err != nil {
		return nil, nil, "", fmt.Errorf("err: %s", err)
	}

	dp := response.GetDeploymentPackage()
	log.Info(fmt.Sprintf("Deployment Package found %v", dp))

	defaultNamespaces := dp.DefaultNamespaces
	log.Info(fmt.Sprintf("Deployment Package default namespaces %v", defaultNamespaces))

	// Build dependency list for each app
	deps := map[string][]string{}
	for _, dep := range dp.ApplicationDependencies {
		deps[dep.Name] = append(deps[dep.Name], dep.Requires)
	}

	// If no profileName is given and DefaultProfileName is defined
	if profileName == "" && dp.DefaultProfileName != "" {
		profileName = dp.DefaultProfileName
	}

	cp := getProfileMap(dp.Profiles, profileName)
	// If no deployment profile matched then stop and error deployment
	// but only check if profile name or default profile is provided.
	if profileName != "" {
		if len(cp) == 0 {
			return nil, nil, "", fmt.Errorf("deployment Profile %s not found", profileName)
		}
	}

	a := []HelmApp{}
	for _, appRef := range dp.GetApplicationReferences() {
		log.Info(fmt.Sprintf("Look up Application Name: %s | Version %s", appRef.Name, appRef.Version))
		appreq := catalog.GetApplicationRequest{
			ApplicationName: appRef.Name,
			Version:         appRef.Version,
		}

		appresp, err := client.GetApplication(ctx, &appreq)
		if err != nil {
			return nil, nil, "", err
		}

		app := appresp.GetApplication()
		log.Info(fmt.Sprintf("Application found: %v", app))

		log.Info(fmt.Sprintf("Look up Helm Registry Name: %s", app.HelmRegistryName))
		regreq := catalog.GetRegistryRequest{
			RegistryName:      app.HelmRegistryName,
			ShowSensitiveInfo: true,
		}

		regresp, err := client.GetRegistry(ctx, &regreq)
		if err != nil {
			return nil, nil, "", err
		}

		helmregistry := regresp.GetRegistry()
		log.Info(fmt.Sprintf("Helm Registry found: %v", helmregistry))

		var imageregistry *catalog.Registry
		if app.ImageRegistryName != "" {
			log.Info(fmt.Sprintf("Look up Image Registry Name: %s", app.ImageRegistryName))
			regreq := catalog.GetRegistryRequest{
				RegistryName:      app.ImageRegistryName,
				ShowSensitiveInfo: true,
			}

			regresp, err := client.GetRegistry(ctx, &regreq)
			if err != nil {
				return nil, nil, "", err
			}

			imageregistry = regresp.GetRegistry()
			log.Info(fmt.Sprintf("Image Registry found: %v", imageregistry))
		}

		values, ok := getAppValues(app, cp)
		if !ok {
			log.Info(fmt.Sprintf("No matching profile found for application %s", app.Name))
		} else if values == "" {
			log.Info(fmt.Sprintf("Application %s has empty values file", app.Name))
		}

		ha := newHelmApp(app, helmregistry, imageregistry, values, cp[app.Name])
		ha.DependsOn = deps[app.Name]

		log.Info(fmt.Sprintf("CatalogLookup App Name: %s | Default Namespace %v", appRef.Name, defaultNamespaces[appRef.Name]))
		ha.DefaultNamespace = defaultNamespaces[appRef.Name]

		for _, appProfile := range app.Profiles {
			if appProfile != nil && appProfile.Name == cp[app.Name] {
				for _, pt := range appProfile.ParameterTemplates {
					name := pt.Name
					displayName := pt.DisplayName
					typeValue := pt.Type
					defaultValue := pt.Default
					mandatory := pt.Mandatory
					secret := pt.Secret

					if displayName == "" {
						displayName = pt.Name
					}

					// Error if mandatory is true and there's a default value
					if mandatory && defaultValue != "" {
						return nil, nil, "", fmt.Errorf("application %s: mandatory parameter "+
							"template %s should have no default value", appRef.Name, displayName)
					}

					// Error if secret is true and there's a default value
					if secret && defaultValue != "" {
						return nil, nil, "", fmt.Errorf("application %s: secret parameter "+
							"template %s should have no default value", appRef.Name, displayName)
					}

					ha.ParameterTemplates = append(ha.ParameterTemplates, ParameterTemplate{
						Name:        name,
						Mandatory:   mandatory,
						Type:        typeValue,
						Secret:      secret,
						DisplayName: displayName,
					})
				}

				for _, dr := range appProfile.DeploymentRequirement {

					request := catalog.GetDeploymentPackageRequest{
						DeploymentPackageName: dr.Name,
						Version:               dr.Version,
					}

					response, err := client.GetDeploymentPackage(ctx, &request)
					if err != nil {
						return nil, nil, "", err
					}

					depProfileName := dr.DeploymentProfileName
					if depProfileName == "" {
						if response.GetDeploymentPackage().DefaultProfileName != "" {
							depProfileName = response.GetDeploymentPackage().DefaultProfileName
						} else {
							return nil, nil, "", fmt.Errorf("profile name and default profile name for dependent deployment package are empty")
						}
					}

					ha.RequiredDeploymentPackages = append(ha.RequiredDeploymentPackages, RequiredDeploymentPackage{
						Name:    dr.Name,
						Version: dr.Version,
						Profile: depProfileName,
					})
				}
			}
		}

		a = append(a, ha)
	}

	helmapps = &a

	return dp, helmapps, profileName, nil
}

// CatalogLookupAPIExtensions retrieves API Extensions associated with a Deployment Package.
var CatalogLookupAPIExtensions = func(ctx context.Context, client CatalogClient, dpName string, dpVersion string) ([]*catalog.APIExtension, error) {
	resp, err := client.GetDeploymentPackage(ctx, &catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: dpName,
		Version:               dpVersion,
	})

	if err != nil {
		// We don't know whether API extension exists or not yet, so don't set condition yet
		// Return with error for another try
		log.Error(err, "reconcileAPIExtension: failed to look up catalog for APIExtension")
		return nil, err
	}

	dp := resp.GetDeploymentPackage()
	return dp.GetExtensions(), nil
}

var CatalogLookupDeploymentPackage = func(ctx context.Context, client CatalogClient, dpName string, dpVersion string) (*catalog.DeploymentPackage, error) {
	log.Info(fmt.Sprintf("Look up Deployment Package Name: %s | Version %s", dpName, dpVersion))
	request := catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: dpName,
		Version:               dpVersion,
	}
	response, err := client.GetDeploymentPackage(ctx, &request)
	if err != nil {
		log.Error(err, "failed to look up deployment package")
		return nil, err
	}
	dp := response.GetDeploymentPackage()
	log.Info(fmt.Sprintf("Deployment Package found: %v", dp))
	return dp, nil
}

var CatalogLookupGrafanaArtifacts = func(ctx context.Context, client CatalogClient, dpName string, dpVersion string) ([]*catalog.Artifact, error) {
	var err error
	grafanaArtifacts := make([]*catalog.Artifact, 0)

	log.Info(fmt.Sprintf("Look up Grafana artifacts: Deployment Package Name %s | Version %s", dpName, dpVersion))

	dpRequest := catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: dpName,
		Version:               dpVersion,
	}
	dpResponse, err := client.GetDeploymentPackage(ctx, &dpRequest)
	if err != nil {
		return nil, err
	}

	artifactRefs := dpResponse.GetDeploymentPackage().GetArtifacts()

	artiResponse, err := client.ListArtifacts(ctx, &catalog.ListArtifactsRequest{})
	if err != nil {
		return nil, err
	}
	artifacts := artiResponse.GetArtifacts()

	for _, ar := range artifactRefs {
		if !strings.EqualFold(ar.Purpose, ArtifactGrafanaPurpose) {
			continue
		}

		for i := 0; i < len(artifacts); i++ {
			if ar.Name == artifacts[i].Name {
				grafanaArtifacts = append(grafanaArtifacts, artifacts[i])
			}
		}
	}

	return grafanaArtifacts, nil
}

var UpdateIsDeployed = func(ctx context.Context, client CatalogClient, dpName string, dpVersion string,
	isDeployed bool) error {

	getReq := catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: dpName,
		Version:               dpVersion,
	}

	log.Info(fmt.Sprintf("Lookup Deployment Package Name: %s", getReq.String()))
	response, err := client.GetDeploymentPackage(ctx, &getReq)
	if err != nil {
		return err
	}
	dp := response.GetDeploymentPackage()

	// Update isDeployed flag if the current value and requested value doesn't match
	if dp.IsDeployed != isDeployed {
		dp.IsDeployed = isDeployed
		updateReq := catalog.UpdateDeploymentPackageRequest{
			DeploymentPackageName: dpName,
			Version:               dpVersion,
			DeploymentPackage:     dp,
		}

		log.Info(fmt.Sprintf("Update Deployment Package Name: %s", updateReq.String()))
		_, err = client.UpdateDeploymentPackage(ctx, &updateReq)
		if err != nil {
			return err
		}
	}

	return nil
}
