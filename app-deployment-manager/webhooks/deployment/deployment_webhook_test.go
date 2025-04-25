// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	catalogmockery "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mockery"
	nbmocks "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
)

var _ = Describe("Deployment Webhook", func() {
	const (
		timeout     = time.Second * 10
		interval    = time.Millisecond * 250
		dpname      = "wordpress"
		dpver_0_1_0 = "0.1.0"
		dpver_0_1_1 = "0.1.1"

		deploymentNamespace = "default"
		deploymentName      = "wordpress-deployment"
	)

	var (
		cc         *catalogmockery.MockeryCatalogClient
		anyContext = mock.AnythingOfType("*context.timerCtx")

		deploymentLookupKey = types.NamespacedName{Name: deploymentName, Namespace: deploymentNamespace}
	)

	When("a new Deployment is created", func() {
		var (
			dpReqGet                     *catalog.GetDeploymentPackageRequest
			dpReqUpdateIsDeployedTrueOld *catalog.UpdateDeploymentPackageRequest
			dpRespIsDeployedFalse        *catalog.GetDeploymentPackageResponse
			deployment                   *v1beta1.Deployment
			vaultAuthMock                *nbmocks.VaultAuth
		)

		BeforeEach(func() {
			deployment = &v1beta1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "app.edge-orchestrator.intel.com/v1beta1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: deploymentNamespace,
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "anyname",
					Project:     "anyproject",
					DeploymentPackageRef: v1beta1.DeploymentPackageRef{
						Name:    dpname,
						Version: dpver_0_1_0,
					},
					Applications:   []v1beta1.Application{},
					DeploymentType: "anytype",
				},
			}

			dpReqGet = &catalog.GetDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_0,
			}

			dpReqUpdateIsDeployedTrueOld = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_0,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: true,
					Name:       dpname,
					Version:    dpver_0_1_0,
				},
			}

			dpRespIsDeployedFalse = &catalog.GetDeploymentPackageResponse{
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: false,
					Name:       dpname,
					Version:    dpver_0_1_0,
				},
			}

			// Ensure the Deployment does not exists
			Expect(k8sClient.Get(ctx, deploymentLookupKey, &v1beta1.Deployment{})).ShouldNot(Succeed())

			cc = catalogmockery.NewMockeryCatalogClient(GinkgoT())
			deploymentwebhook.catalogclient = cc

			// M2M auth client mock
			vaultAuthMock = &nbmocks.VaultAuth{}
			deploymentwebhook.vaultAuthClient = vaultAuthMock

			vaultAuthMock.On("GetM2MToken", nbmocks.AnyContextValue).Return("test-m2m-token", nil)
			cc.On("GetDeploymentPackage", anyContext, dpReqGet).Return(dpRespIsDeployedFalse, nil)
			cc.On("UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedTrueOld).Return(&emptypb.Empty{}, nil)
		})

		AfterEach(func() {
			// Delete the Deployment so that DeletionTimestamp to be set
			Expect(k8sClient.DeleteAllOf(ctx, &v1beta1.Deployment{}, client.InNamespace(deploymentNamespace))).To(Succeed())

			// Remove finalizers in webhook test as it's handled by deployment controller
			createdDeployment := &v1beta1.Deployment{}
			Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerGitRemote)))
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerCatalog)))
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerDependency)))
			Expect(k8sClient.Update(ctx, createdDeployment)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, &v1beta1.Deployment{})
				return err == nil
			}, timeout, interval).Should(BeFalse())

			cc.AssertExpectations(GinkgoT())
		})

		It("should set finalizer app.edge-orchestrator.intel.com/catalog", func() {
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			createdDeployment := &v1beta1.Deployment{}
			Expect(cutil.ContainsFinalizer(createdDeployment, string(v1beta1.FinalizerCatalog)))
		})

		It("should request Catalog service to set isDeployed to True for the deployment package", func() {
			// Create deployment package
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			createdDeployment := &v1beta1.Deployment{}
			Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())

			// Assert calls to the catalog service
			cc.AssertCalled(GinkgoT(), "UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedTrueOld)
		})
	})

	When("a deployment package version is updated", func() {
		var (
			deployment                    *v1beta1.Deployment
			deploymentExisting            *v1beta1.Deployment
			dpReqGetOld                   *catalog.GetDeploymentPackageRequest
			dpReqGetNew                   *catalog.GetDeploymentPackageRequest
			dpReqUpdateIsDeployedTrueNew  *catalog.UpdateDeploymentPackageRequest
			dpReqUpdateIsDeployedFalseOld *catalog.UpdateDeploymentPackageRequest
			dpRespIsDeployedFalseOld      *catalog.GetDeploymentPackageResponse
			dpReqUpdateIsDeployedTrueOld  *catalog.UpdateDeploymentPackageRequest
			vaultAuthMock                 *nbmocks.VaultAuth
		)

		BeforeEach(func() {
			deployment = &v1beta1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "app.edge-orchestrator.intel.com/v1beta1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: deploymentNamespace,
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "anyname",
					Project:     "anyproject",
					DeploymentPackageRef: v1beta1.DeploymentPackageRef{
						Name:    dpname,
						Version: dpver_0_1_0,
					},
					Applications:   []v1beta1.Application{},
					DeploymentType: "anytype",
				},
			}

			deploymentExisting = &v1beta1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "app.edge-orchestrator.intel.com/v1beta1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-wordpress-existing",
					Namespace: deploymentNamespace,
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "anyname",
					Project:     "anyproject",
					DeploymentPackageRef: v1beta1.DeploymentPackageRef{
						Name:    dpname,
						Version: dpver_0_1_0,
					},
					Applications:   []v1beta1.Application{},
					DeploymentType: "anytype",
				},
			}

			dpReqGetOld = &catalog.GetDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_0,
			}

			dpReqGetNew = &catalog.GetDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_1,
			}

			dpReqUpdateIsDeployedFalseOld = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_0,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: false,
					Name:       dpname,
					Version:    dpver_0_1_0,
				},
			}

			dpReqUpdateIsDeployedTrueNew = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_1,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: true,
					Name:       dpname,
					Version:    dpver_0_1_1,
				},
			}

			dpReqUpdateIsDeployedTrueOld = &catalog.UpdateDeploymentPackageRequest{
				DeploymentPackageName: dpname,
				Version:               dpver_0_1_0,
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: true,
					Name:       dpname,
					Version:    dpver_0_1_0,
				},
			}

			dpRespIsDeployedFalseOld = &catalog.GetDeploymentPackageResponse{
				DeploymentPackage: &catalog.DeploymentPackage{
					IsDeployed: false,
					Name:       dpname,
					Version:    dpver_0_1_0,
				},
			}

			// Ensure the Deployment does not exists
			Expect(k8sClient.Get(ctx, deploymentLookupKey, &v1beta1.Deployment{})).ShouldNot(Succeed())

			cc = catalogmockery.NewMockeryCatalogClient(GinkgoT())
			deploymentwebhook.catalogclient = cc

			// M2M auth client mock
			vaultAuthMock = &nbmocks.VaultAuth{}
			deploymentwebhook.vaultAuthClient = vaultAuthMock

			vaultAuthMock.On("GetM2MToken", nbmocks.AnyContextValue).Return("test-m2m-token", nil)
			cc.On("GetDeploymentPackage", anyContext, dpReqGetOld).Return(dpRespIsDeployedFalseOld, nil)
			cc.On("UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedTrueOld).Return(&emptypb.Empty{}, nil)

			// Create a deployment to be updated in the following tests
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			createdDeployment := &v1beta1.Deployment{}
			Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
		})

		AfterEach(func() {
			// Delete the Deployment so that DeletionTimestamp to be set
			Expect(k8sClient.DeleteAllOf(ctx, &v1beta1.Deployment{}, client.InNamespace(deploymentNamespace))).To(Succeed())

			// Remove finalizers in webhook test as it's handled by deployment controller
			createdDeployment := &v1beta1.Deployment{}
			Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerGitRemote)))
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerDependency)))
			Expect(cutil.RemoveFinalizer(createdDeployment, string(v1beta1.FinalizerCatalog)))
			Expect(k8sClient.Update(ctx, createdDeployment)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, &v1beta1.Deployment{})
				return err == nil
			}, timeout, interval).Should(BeFalse())
		})

		Context("update spec is invalid", func() {
			Context("app name was updated", func() {
				It("should return error", func() {
					createdDeployment := &v1beta1.Deployment{}
					Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
					createdDeployment.Spec.DeploymentPackageRef.Name = "new_app_name"
					Expect(k8sClient.Update(ctx, createdDeployment)).ShouldNot(Succeed())
				})
			})
		})

		Context("update spec is valid", func() {
			It("should set isDeployed for the new deployment package in Catalog", func() {
				cc.On("GetDeploymentPackage", anyContext, dpReqGetNew).Return(dpRespIsDeployedFalseOld, nil)
				cc.On("UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedFalseOld).Return(&emptypb.Empty{}, nil)

				// Update the deployment
				createdDeployment := &v1beta1.Deployment{}
				Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
				createdDeployment.Spec.DeploymentPackageRef.Version = dpver_0_1_1
				Expect(k8sClient.Update(ctx, createdDeployment)).To(Succeed())

				// Assert calls to the catalog service
				cc.AssertCalled(GinkgoT(), "GetDeploymentPackage", anyContext, dpReqGetNew)
				cc.AssertCalled(GinkgoT(), "UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedFalseOld)
			})
		})

		Context("there are no other deployments using the old deployment package version", func() {
			It("should unset isDeployed for the old version in the Catalog", func() {
				cc.On("GetDeploymentPackage", anyContext, dpReqGetNew).Return(dpRespIsDeployedFalseOld, nil)
				cc.On("UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedFalseOld).Return(&emptypb.Empty{}, nil)

				// Update the deployment
				createdDeployment := &v1beta1.Deployment{}
				Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
				createdDeployment.Spec.DeploymentPackageRef.Version = dpver_0_1_1
				Expect(k8sClient.Update(ctx, createdDeployment)).To(Succeed())

				// Assert calls to the catalog service
				cc.AssertCalled(GinkgoT(), "GetDeploymentPackage", anyContext, dpReqGetNew)
				cc.AssertCalled(GinkgoT(), "UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedFalseOld)
			})
		})

		Context("there are other deployments using the old deployment package version", func() {
			It("should not unset isDeployed flag for the old deployment package in Catalog", func() {
				cc.On("GetDeploymentPackage", anyContext, dpReqGetNew).Return(dpRespIsDeployedFalseOld, nil)

				// Create an exisiting deployment that uses the same version as the old deployment
				Expect(k8sClient.Create(ctx, deploymentExisting)).To(Succeed())
				Expect(k8sClient.Get(ctx, deploymentLookupKey, &v1beta1.Deployment{})).To(Succeed())

				// Update the deployment
				createdDeployment := &v1beta1.Deployment{}
				Expect(k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)).To(Succeed())
				createdDeployment.Spec.DeploymentPackageRef.Version = dpver_0_1_1
				Expect(k8sClient.Update(ctx, createdDeployment)).To(Succeed())

				// Assert calls to the catalog service
				cc.AssertNotCalled(GinkgoT(), "UpdateDeploymentPackage", anyContext, dpReqUpdateIsDeployedTrueNew)
			})
		})

	})
})
