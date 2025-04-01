// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package catalogclient_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	mockerymock "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mockery"
	mocks "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mocks"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("CatalogClient tests", func() {
	const (
		dpname    = "wordpress"
		dpnamebad = "doesnotexist"
		dpver     = "0.1.0"
		dpprof    = "default"

		appname   = "wordpress"
		appver    = "1.0.0"
		appprof   = "default"
		appvalues = `{"foo": "bar"}`

		chartname = "wordpress"
		chartver  = "15.2.42"

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
		expectedHelmApps []catalogclient.HelmApp

		appRef  catalog.ApplicationReference
		appReq  catalog.GetApplicationRequest
		appResp catalog.GetApplicationResponse

		dpReqGood  catalog.GetDeploymentPackageRequest
		dpRespGood catalog.GetDeploymentPackageResponse
		dpReqBad   catalog.GetDeploymentPackageRequest

		helmRegReq    catalog.GetRegistryRequest
		helmRegResp   catalog.GetRegistryResponse
		dockerRegReq  catalog.GetRegistryRequest
		dockerRegResp catalog.GetRegistryResponse
	)

	BeforeEach(func() {
		appRef = catalog.ApplicationReference{
			Name:    appname,
			Version: appver,
		}

		// Deployment package mock request and response
		dpReqGood = catalog.GetDeploymentPackageRequest{
			DeploymentPackageName: dpname,
			Version:               dpver,
		}
		dpRespGood = catalog.GetDeploymentPackageResponse{
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
				Extensions:        []*catalog.APIExtension{},
				Artifacts:         []*catalog.ArtifactReference{},
				DefaultNamespaces: map[string]string{},
			},
		}
		dpReqBad = catalog.GetDeploymentPackageRequest{
			DeploymentPackageName: dpnamebad,
			Version:               dpver,
		}

		// Application mock request and response
		appReq = catalog.GetApplicationRequest{
			ApplicationName: appname,
			Version:         appver,
		}
		appResp = catalog.GetApplicationResponse{
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

		// Registry mock request and response
		helmRegReq = catalog.GetRegistryRequest{
			RegistryName:      helmreg,
			ShowSensitiveInfo: true,
		}
		helmRegResp = catalog.GetRegistryResponse{
			Registry: &catalog.Registry{
				Name:      helmreg,
				RootUrl:   helmurl,
				Username:  helmuser,
				AuthToken: helmpass,
				Type:      "HELM",
				Cacerts:   helmcacert,
			},
		}
		dockerRegReq = catalog.GetRegistryRequest{
			RegistryName:      dockerreg,
			ShowSensitiveInfo: true,
		}
		dockerRegResp = catalog.GetRegistryResponse{
			Registry: &catalog.Registry{
				Name:      dockerreg,
				RootUrl:   dockerurl,
				Username:  dockeruser,
				AuthToken: dockerpass,
				Type:      "IMAGE",
			},
		}

		expectedHelmApps = []catalogclient.HelmApp{
			{
				Name:                dpname,
				Repo:                helmurl,
				ImageRegistry:       dockerurl,
				Chart:               chartname,
				Version:             chartver,
				Profile:             appprof,
				Values:              appvalues,
				DependsOn:           []string{"dependency"},
				RedeployAfterUpdate: false,
				DefaultNamespace:    "",
				HelmCredential: catalogclient.HelmCredential{
					Username: helmuser,
					Password: helmpass,
					CaCerts:  helmcacert,
				},
				DockerCredential: catalogclient.DockerCredential{
					Username: dockeruser,
					Password: dockerpass,
				},
				ParameterTemplates: []catalogclient.ParameterTemplate{},
			},
		}
	})

	When("looking up an existing deployment package", func() {
		Context("and no profile is specified", func() {
			It("should succeed and return the default profile", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil).Times(2)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, ha, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, "")
				Expect(ha).To(Equal(&expectedHelmApps))
				Expect(defaultprof).To(Equal(dpprof))
				Expect(err).To(BeNil())

				apie, err := catalogclient.CatalogLookupAPIExtensions(context.Background(), mockCatalog, dpname, dpver)
				Expect(apie).To(BeEmpty())
				Expect(err).To(BeNil())
			})
		})

		Context("and a profile is specified", func() {
			It("should succeed and return the profile specified", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, ha, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, dpprof)
				Expect(ha).To(Equal(&expectedHelmApps))
				Expect(defaultprof).To(Equal(dpprof))
				Expect(err).To(BeNil())
			})
		})

		Context("and the application lookup fails", func() {
			It("should return an error", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(nil, errors.New("App lookup failed"))

				_, _, _, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, dpprof)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("and the Helm registry lookup fails", func() {
			It("should return an error", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(nil, errors.New("Helm registry lookup failed"))
				// mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, _, _, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, dpprof)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("and the Docker registry lookup fails", func() {
			It("should return an error", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(nil, errors.New("Docker registry lookup failed"))

				_, _, _, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, dpprof)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("and app profile name does not match existing profile", func() {
			It("should succeed and return an empty app values", func() {
				appResp.Application.Profiles = []*catalog.Profile{}
				expectedHelmApps[0].Values = ""

				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, ha, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, "")
				Expect(ha).To(Equal(&expectedHelmApps))
				Expect(defaultprof).To(Equal(dpprof))
				Expect(err).To(BeNil())
			})
		})

		Context("and app profile name matches but is empty", func() {
			It("should succeed and return an empty app values", func() {
				appResp.Application.Profiles = []*catalog.Profile{
					{
						Name:        appprof,
						ChartValues: "",
					},
				}
				expectedHelmApps[0].Values = ""

				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, ha, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, "")
				Expect(ha).To(Equal(&expectedHelmApps))
				Expect(defaultprof).To(Equal(dpprof))
				Expect(err).To(BeNil())
			})
		})

		Context("and deployment package has no default profile and lists no profiles", func() {
			It("should succeed", func() {
				dpRespGood.DeploymentPackage.Profiles = []*catalog.DeploymentProfile{}
				dpRespGood.DeploymentPackage.DefaultProfileName = ""
				expectedHelmApps[0].Profile = ""
				expectedHelmApps[0].Values = ""

				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)
				mockCatalog.EXPECT().GetApplication(gomock.Any(), &appReq).Return(&appResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &helmRegReq).Return(&helmRegResp, nil)
				mockCatalog.EXPECT().GetRegistry(gomock.Any(), &dockerRegReq).Return(&dockerRegResp, nil)

				_, ha, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, "")
				Expect(ha).To(Equal(&expectedHelmApps))
				Expect(defaultprof).To(Equal(""))
				Expect(err).To(BeNil())
			})
		})

		Context("and provided profile does not match deployment package profiles", func() {
			It("should return an error", func() {
				mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
				mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqGood).Return(&dpRespGood, nil)

				_, _, defaultprof, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpname, dpver, "test-profilename")
				Expect(defaultprof).To(Equal(""))
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).Should(Equal("deployment Profile test-profilename not found"))
			})
		})
	})

	When("looking up a nonexisting deployment package", func() {
		It("should return an error", func() {
			mockCatalog := mocks.NewMockCatalogClient(mockCtrl)
			mockCatalog.EXPECT().GetDeploymentPackage(gomock.Any(), &dpReqBad).Return(nil, errors.New("Lookup failed")).Times(2)

			_, _, _, err := catalogclient.CatalogLookupDPAndHelmApps(context.Background(), mockCatalog, dpnamebad, dpver, "")
			Expect(err).ToNot(BeNil())

			_, err = catalogclient.CatalogLookupAPIExtensions(context.Background(), mockCatalog, dpnamebad, dpver)
			Expect(err).ToNot(BeNil())
		})
	})

	When("calling Connect function", func() {
		It("should succeed", func() {
			_, err := catalogclient.Connect(context.Background(), "0.0.0.0")
			Expect(err).To(BeNil())
		})
	})

	When("calling NewCatalogClient function", func() {
		Context("if CATALOG_SERVICE_ADDRESS environment value is not set", func() {
			It("shoud return error", func() {
				os.Unsetenv("CATALOG_SERVICE_ADDRESS")
				_, err := catalogclient.NewCatalogClient()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).Should(Equal("catalog service address is not set"))
			})
		})
	})

	When("updating IsDeployed flag for a Deployment Package", func() {
		var (
			dpReqUpdateIsDeployedTrue  *catalog.UpdateDeploymentPackageRequest
			dpReqUpdateIsDeployedFalse *catalog.UpdateDeploymentPackageRequest
			dpRespIsDeployedTrue       *catalog.GetDeploymentPackageResponse
			dpRespIsDeployedFalse      *catalog.GetDeploymentPackageResponse
			anyReqGet                  *catalog.GetDeploymentPackageRequest
		)

		BeforeEach(func() {
			anyReqGet = &catalog.GetDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver,
			}

			dpReqUpdateIsDeployedTrue = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: true,
					Name:       dpname,
					Version:    dpver,
				},
			}

			dpReqUpdateIsDeployedFalse = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: false,
					Name:       dpname,
					Version:    dpver,
				},
			}

			dpRespIsDeployedTrue = &catalog.GetDeploymentPackageResponse{
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: true,
					Name:       dpname,
					Version:    dpver,
				},
			}

			dpRespIsDeployedFalse = &catalog.GetDeploymentPackageResponse{
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: false,
					Name:       dpname,
					Version:    dpver,
				},
			}
		})

		Context("isDeployed for the test deployment package is set to False in Catalog", func() {
			Context("setting isDeployed to True", func() {
				It("should request update", func() {
					cc := mockerymock.NewMockeryCatalogClient(GinkgoT())

					cc.On("GetDeploymentPackage", context.TODO(), anyReqGet).Return(dpRespIsDeployedFalse, nil)
					cc.On("UpdateDeploymentPackage", context.TODO(), dpReqUpdateIsDeployedTrue).Return(&emptypb.Empty{}, nil)

					Expect(catalogclient.UpdateIsDeployed(context.TODO(), cc, dpname, dpver, true)).To(Succeed())
					cc.AssertExpectations(GinkgoT())
				})
			})
			Context("setting isDeployed to False", func() {
				It("should not request update", func() {
					cc := mockerymock.NewMockeryCatalogClient(GinkgoT())
					cc.On("GetDeploymentPackage", context.TODO(), anyReqGet).Return(dpRespIsDeployedFalse, nil)

					Expect(catalogclient.UpdateIsDeployed(context.TODO(), cc, dpname, dpver, false)).To(Succeed())
					cc.AssertExpectations(GinkgoT())
				})
			})
		})
		Context("isDeployed for the test deployment package is set to True in Catalog", func() {
			Context("setting isDeployed to True", func() {
				It("should not request update", func() {
					cc := mockerymock.NewMockeryCatalogClient(GinkgoT())
					cc.On("GetDeploymentPackage", context.TODO(), anyReqGet).Return(dpRespIsDeployedTrue, nil)

					Expect(catalogclient.UpdateIsDeployed(context.TODO(), cc, dpname, dpver, true)).To(Succeed())
					cc.AssertExpectations(GinkgoT())
				})
			})
			Context("setting isDeployed to False", func() {
				It("should request update", func() {
					cc := mockerymock.NewMockeryCatalogClient(GinkgoT())
					cc.On("GetDeploymentPackage", context.TODO(), anyReqGet).Return(dpRespIsDeployedTrue, nil)
					cc.On("UpdateDeploymentPackage", context.TODO(), dpReqUpdateIsDeployedFalse).Return(&emptypb.Empty{}, nil)

					Expect(catalogclient.UpdateIsDeployed(context.TODO(), cc, dpname, dpver, false)).To(Succeed())
					cc.AssertExpectations(GinkgoT())
				})
			})
		})
	})
})

func TestCatalogLookupDeploymentPackage(t *testing.T) {
	md := metadata.Pairs("foo", "test")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: "deployment-package",
		Version:               "version",
	}
	resp := &catalog.GetDeploymentPackageResponse{
		DeploymentPackage: &catalog.DeploymentPackage{
			Name:    "deployment-package",
			Version: "version",
		},
	}

	cc := mockerymock.NewMockeryCatalogClient(t)
	cc.On("GetDeploymentPackage", mock.AnythingOfType("*context.valueCtx"), req).Return(resp, nil)

	dp, err := catalogclient.CatalogLookupDeploymentPackage(ctx, cc, "deployment-package", "version")
	assert.NoError(t, err)
	assert.NotNil(t, dp)
}

func TestCatalogLookupGrafanaArtifacts(t *testing.T) {
	md := metadata.Pairs("foo", "test")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	dpReq := &catalog.GetDeploymentPackageRequest{
		DeploymentPackageName: "deployment-package",
		Version:               "version",
	}
	dpResp := &catalog.GetDeploymentPackageResponse{
		DeploymentPackage: &catalog.DeploymentPackage{
			Name:    "deployment-package",
			Version: "version",
			Artifacts: []*catalog.ArtifactReference{
				{
					Name:    "sample-dashboard",
					Purpose: "grafana",
				},
			},
		},
	}
	artiReq := &catalog.ListArtifactsRequest{}
	artiResp := &catalog.ListArtifactsResponse{
		Artifacts: []*catalog.Artifact{
			{
				Name:     "sample-dashboard",
				MimeType: "application/json",
				Artifact: []byte("JSON_CONTENTS"),
			},
		},
	}
	cc := mockerymock.NewMockeryCatalogClient(t)
	cc.On("GetDeploymentPackage", mock.AnythingOfType("*context.valueCtx"), dpReq).Return(dpResp, nil)
	cc.On("ListArtifacts", mock.AnythingOfType("*context.valueCtx"), artiReq).Return(artiResp, nil)

	artifacts, err := catalogclient.CatalogLookupGrafanaArtifacts(ctx, cc, "deployment-package", "version")
	assert.NoError(t, err)
	assert.NotNil(t, artifacts)

}
