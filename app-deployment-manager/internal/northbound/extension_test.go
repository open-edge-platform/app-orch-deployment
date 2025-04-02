// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"os"

	"errors"
	"github.com/bufbuild/protovalidate-go"
	"net/http"
	"net/http/httptest"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mockerymock "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mockery"
	nbmocks "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var _ = Describe("Gateway gRPC Service", func() {
	var (
		deploymentServer     *DeploymentSvc
		k8sClient            *mocks.FakeDeploymentV1
		protoValidator       *protovalidate.Validator
		apiExtensionInstance *deploymentv1beta1.APIExtension
		apiExtensionListSrc  deploymentv1beta1.APIExtensionList
	)

	Describe("APIExtension Get", func() {
		BeforeEach(func() {
			os.Setenv("API_EXT_ENABLED", "true")

			// populates a mock APIExtension object
			var apiExtensionListSrc deploymentv1beta1.APIExtensionList
			setAPIExtensionListObject(&apiExtensionListSrc)

			apiExtensionInstance = setAPIExtensionInstance(&apiExtensionListSrc)

			k8sClient = &mocks.FakeDeploymentV1{}

			// protovalidate Validator
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, protoValidator, nil)

			k8sClient.On(
				"Get", context.Background(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, nil)

		})

		It("successfully get a APIExtension", func() {
			resp, err := deploymentServer.GetAPIExtension(context.Background(), &deploymentpb.GetAPIExtensionRequest{
				Name: "test",
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ApiExtension.Token).To(Equal("this-is-a-test-token"))
		})

		It("fails due to API extensions is disabled", func() {
			os.Setenv("API_EXT_ENABLED", "false")
			resp, err := deploymentServer.GetAPIExtension(context.Background(), &deploymentpb.GetAPIExtensionRequest{
				Name: "test",
			})

			Expect(resp).Should(BeNil())
			Expect(err).Should(HaveOccurred())
		})

		It("fails due to missing name", func() {
			_, err := deploymentServer.GetAPIExtension(context.Background(), &deploymentpb.GetAPIExtensionRequest{
				Name: "",
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.InvalidArgument))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("incomplete request"))
		})

		It("fails due to API_EXT_ENABLED env not set", func() {
			os.Unsetenv("API_EXT_ENABLED")
			_, err := deploymentServer.GetAPIExtension(context.Background(), &deploymentpb.GetAPIExtensionRequest{
				Name: "test",
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("API_EXT_ENABLED env var not set"))
		})

		It("fails due to GET error", func() {
			k8sClient := &mocks.FakeDeploymentV1{}

			// protovalidate Validator
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, protoValidator, nil)

			k8sClient.On(
				"Get", context.Background(), mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, errors.New("mock err"))

			_, err = deploymentServer.GetAPIExtension(context.Background(), &deploymentpb.GetAPIExtensionRequest{
				Name: "test",
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})
	})

	Describe("UIExtension List", func() {
		BeforeEach(func() {
			os.Setenv("API_EXT_ENABLED", "true")

			// populates a mock APIExtension object
			setAPIExtensionListObject(&apiExtensionListSrc)

			apiExtensionInstance = setAPIExtensionInstance(&apiExtensionListSrc)

			k8sClient = &mocks.FakeDeploymentV1{}
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, protoValidator, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.APIExtensionList{
				ListMeta: apiExtensionListSrc.ListMeta,
				TypeMeta: apiExtensionListSrc.TypeMeta,
				Items:    apiExtensionListSrc.Items,
			}, nil)

		})

		It("successfully returns a list of UIExtension", func() {
			resp, err := deploymentServer.ListUIExtensions(context.Background(), &deploymentpb.ListUIExtensionsRequest{
				ServiceName: []string{"test-servicename"},
			})

			Expect(err).Should(Succeed())
			Expect(len(resp.UiExtensions)).To(Equal(1))
			Expect(resp.UiExtensions[0].ServiceName).To(Equal("test-servicename"))
		})

		It("fails due to API extensions is disabled", func() {
			os.Setenv("API_EXT_ENABLED", "false")
			resp, err := deploymentServer.ListUIExtensions(context.Background(), &deploymentpb.ListUIExtensionsRequest{
				ServiceName: []string{"test-servicename"},
			})

			Expect(resp).Should(BeNil())
			Expect(err).Should(HaveOccurred())
		})

		It("fails due to API_EXT_ENABLED env not set", func() {
			os.Unsetenv("API_EXT_ENABLED")
			_, err := deploymentServer.ListUIExtensions(context.Background(), &deploymentpb.ListUIExtensionsRequest{
				ServiceName: []string{"test-servicename"},
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.NotFound))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("API_EXT_ENABLED env var not set"))
		})

		It("fails due to LIST error", func() {
			k8sClient := &mocks.FakeDeploymentV1{}
			protoValidator, err := protovalidate.New()
			Expect(err).ToNot(HaveOccurred())

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, protoValidator, nil)

			k8sClient.On(
				"List", context.Background(), mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.APIExtensionList{
				ListMeta: apiExtensionListSrc.ListMeta,
				TypeMeta: apiExtensionListSrc.TypeMeta,
				Items:    apiExtensionListSrc.Items,
			}, errors.New("mock err"))

			_, err = deploymentServer.ListUIExtensions(context.Background(), &deploymentpb.ListUIExtensionsRequest{
				ServiceName: []string{"test-servicename"},
			})

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
			Expect(s.Message()).Should(Equal("mock err"))
		})
	})

	Describe("Delete API Extension CR", func() {
		var (
			ctx                 context.Context
			apiExtensionListSrc deploymentv1beta1.APIExtensionList
			d                   *Deployment
		)

		BeforeEach(func() {
			os.Setenv("API_EXT_ENABLED", "true")

			// populates a mock extension objects
			setAPIExtListObject(&apiExtensionListSrc)

			apiExtensionInstance = setAPIExtensionInstance(&apiExtensionListSrc)

			k8sClient = &mocks.FakeDeploymentV1{}

			md := metadata.Pairs("foo", "test")
			ctx = metadata.NewIncomingContext(context.Background(), md)

			d = setDeployment()
		})

		It("successfully delete api extension CR", func() {
			k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.APIExtensionList{
				ListMeta: apiExtensionListSrc.ListMeta,
				TypeMeta: apiExtensionListSrc.TypeMeta,
				Items:    apiExtensionListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Delete", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(nil).Once()

			err := deleteAPIExtCrs(ctx, k8sClient, d)

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to API_EXT_ENABLED env not set", func() {
			os.Unsetenv("API_EXT_ENABLED")

			err := deleteAPIExtCrs(ctx, nil, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("API_EXT_ENABLED env var not set"))
		})

		It("fails due to LIST error", func() {
			k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.APIExtensionList{
				ListMeta: apiExtensionListSrc.ListMeta,
				TypeMeta: apiExtensionListSrc.TypeMeta,
				Items:    apiExtensionListSrc.Items,
			}, errors.New("mock err")).Once()

			err := deleteAPIExtCrs(ctx, k8sClient, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("mock err"))
		})

		It("fails due to DELETE error", func() {
			k8sClient.On(
				"List", nbmocks.AnyContextValue, mock.AnythingOfType("v1.ListOptions"),
			).Return(&deploymentv1beta1.APIExtensionList{
				ListMeta: apiExtensionListSrc.ListMeta,
				TypeMeta: apiExtensionListSrc.TypeMeta,
				Items:    apiExtensionListSrc.Items,
			}, nil).Once()

			k8sClient.On(
				"Delete", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions"),
			).Return(errors.New("mock err")).Once()

			err := deleteAPIExtCrs(ctx, k8sClient, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("mock err"))
		})
	})

	Describe("Create API Extension CR", func() {
		var (
			catalogClient       *mockerymock.MockeryCatalogClient
			ctx                 context.Context
			apiExtensionListSrc deploymentv1beta1.APIExtensionList
			d                   *Deployment
		)

		BeforeEach(func() {
			os.Setenv("API_EXT_ENABLED", "true")

			// populates a mock extension objects
			setAPIExtensionListObject(&apiExtensionListSrc)

			apiExtensionInstance = setAPIExtensionInstance(&apiExtensionListSrc)

			k8sClient = &mocks.FakeDeploymentV1{}
			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, nil, protoValidator, nil)

			md := metadata.Pairs("foo", "test")
			ctx = metadata.NewIncomingContext(context.Background(), md)

			d = setDeployment()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			catalogClient = mockerymock.NewMockeryCatalogClient(GinkgoT())
		})

		It("successfully create api extension CR", func() {
			k8sClient.On(
				"Get", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, nil).Once()

			k8sClient.On(
				"Create", nbmocks.AnyContextValue, apiExtensionInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(apiExtensionInstance, nil).Once()

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, catalogClient, protoValidator, nil)

			catalogClient.On("GetDeploymentPackage", nbmocks.AnyContextValue, nbmocks.AnyGetDpReq).Return(&nbmocks.DpAPIRespGood, nil)

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to API_EXT_ENABLED env not set", func() {
			os.Unsetenv("API_EXT_ENABLED")

			deploymentServer = NewDeployment(nil, nil, nil, nil, nil, protoValidator, nil)

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("API_EXT_ENABLED env var not set"))
		})

		It("successfully return with no api extension found to create", func() {
			deploymentServer = NewDeployment(nil, nil, nil, nil, catalogClient, protoValidator, nil)

			catalogClient.On("GetDeploymentPackage", nbmocks.AnyContextValue, nbmocks.AnyGetDpReq).Return(&nbmocks.DpNoAPIRespGood, nil)

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due GET error", func() {
			k8sClient.On(
				"Get", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, status.Error(codes.Unknown, "mock err")).Once()

			k8sClient.On(
				"Create", nbmocks.AnyContextValue, apiExtensionInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(apiExtensionInstance, nil).Once()

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, catalogClient, protoValidator, nil)

			catalogClient.On("GetDeploymentPackage", nbmocks.AnyContextValue, nbmocks.AnyGetDpReq).Return(&nbmocks.DpAPIRespGood, nil)

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
		})

		It("fails due CREATE error", func() {
			k8sClient.On(
				"Get", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, nil).Once()

			k8sClient.On(
				"Create", nbmocks.AnyContextValue, apiExtensionInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(apiExtensionInstance, status.Error(codes.Unknown, "mock err")).Once()

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, catalogClient, protoValidator, nil)

			catalogClient.On("GetDeploymentPackage", nbmocks.AnyContextValue, nbmocks.AnyGetDpReq).Return(&nbmocks.DpAPIRespGood, nil)

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
		})

		It("fails due to DeploymentPackage error", func() {
			k8sClient.On(
				"Get", nbmocks.AnyContextValue, mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
			).Return(apiExtensionInstance, nil).Once()

			k8sClient.On(
				"Create", nbmocks.AnyContextValue, apiExtensionInstance, mock.AnythingOfType("v1.CreateOptions"),
			).Return(apiExtensionInstance, nil).Once()

			deploymentServer = NewDeployment(k8sClient, nil, nil, nil, catalogClient, protoValidator, nil)

			catalogClient.On("GetDeploymentPackage", nbmocks.AnyContextValue, nbmocks.AnyGetDpReq).Return(&nbmocks.DpAPIRespGood, status.Error(codes.Unknown, "mock err"))

			err := createAPIExtCrs(ctx, deploymentServer, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeTrue())
		})
	})

})

func setAPIExtensionListObject(apiExtensionListSrc *deploymentv1beta1.APIExtensionList) {
	apiExtensionListSrc.TypeMeta.Kind = KIND_API_EXT
	apiExtensionListSrc.TypeMeta.APIVersion = apiVersion

	apiExtensionListSrc.ListMeta.ResourceVersion = "6"
	apiExtensionListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	apiExtensionListSrc.ListMeta.RemainingItemCount = &remainingItem

	apiExtensionListSrc.Items = make([]deploymentv1beta1.APIExtension, 1)
	setAPIExtensionObject(&apiExtensionListSrc.Items[0])
}

func setAPIExtensionObject(apiExtensionSrc *deploymentv1beta1.APIExtension) {
	apiExtensionSrc.ObjectMeta.Name = "test-deployment"
	apiExtensionSrc.ObjectMeta.GenerateName = "test-generate-name"
	apiExtensionSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	apiExtensionSrc.ObjectMeta.UID = types.UID(VALID_UID)
	apiExtensionSrc.ObjectMeta.ResourceVersion = "6"
	apiExtensionSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	apiExtensionSrc.ObjectMeta.CreationTimestamp = currentTime
	apiExtensionSrc.ObjectMeta.DeletionTimestamp = &currentTime

	apiExtensionSrc.ObjectMeta.Labels = make(map[string]string)
	apiExtensionSrc.ObjectMeta.Labels["app.edge-orchestrator.intel.com/deployment-id"] = ""

	apiExtensionSrc.Spec.DisplayName = "test-displayname"
	apiExtensionSrc.Spec.Project = "test-project"
	apiExtensionSrc.Spec.APIGroup = deploymentv1beta1.APIGroup{
		Name:    "test-name",
		Version: "test-version",
	}
	apiExtensionSrc.Spec.AgentClusterLabels = make(map[string]string)
	apiExtensionSrc.Spec.AgentClusterLabels["color"] = "blue"

	apiExtensionSrc.Spec.ProxyEndpoints = []deploymentv1beta1.ProxyEndpoint{
		{
			ServiceName: "test-servicename",
			Path:        "test-externalpath",
			Backend:     "test-internalpath",
			Scheme:      "test-scheme",
			AuthType:    "test-authtype",
			AppName:     "test-appname",
		},
	}

	apiExtensionSrc.Spec.UIExtensions = []deploymentv1beta1.UIExtension{
		{
			ServiceName: "test-servicename",
			Description: "test-description",
			Label:       "test-label",
			FileName:    "test-filename",
			AppName:     "test-appname",
			ModuleName:  "test-modulename",
		},
	}

	apiExtensionSrc.Status.State = deploymentv1beta1.Running
	apiExtensionSrc.Status.TokenSecretRef = deploymentv1beta1.TokenSecretRef{
		Name:           "test-token",
		GeneratedToken: "this-is-a-test-token",
	}
}

func setAPIExtensionInstance(apiExtensionListSrc *deploymentv1beta1.APIExtensionList) *deploymentv1beta1.APIExtension {
	instance := &deploymentv1beta1.APIExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiExtensionListSrc.Items[0].ObjectMeta.Name,
			Namespace: apiExtensionListSrc.Items[0].ObjectMeta.Namespace,
			Labels:    apiExtensionListSrc.Items[0].ObjectMeta.Labels,
		},
		Spec: deploymentv1beta1.APIExtensionSpec{
			DisplayName: apiExtensionListSrc.Items[0].Spec.DisplayName,
			Project:     apiExtensionListSrc.Items[0].Spec.Project,
			APIGroup:    apiExtensionListSrc.Items[0].Spec.APIGroup,
			ProxyEndpoints: []deploymentv1beta1.ProxyEndpoint{
				{
					ServiceName: apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].ServiceName,
					Path:        apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].Path,
					Backend:     apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].Backend,
					Scheme:      apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].Scheme,
					AuthType:    apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].AuthType,
					AppName:     apiExtensionListSrc.Items[0].Spec.ProxyEndpoints[0].AppName,
				},
			},
			UIExtensions: []deploymentv1beta1.UIExtension{
				{
					ServiceName: apiExtensionListSrc.Items[0].Spec.UIExtensions[0].ServiceName,
					Description: apiExtensionListSrc.Items[0].Spec.UIExtensions[0].Description,
					Label:       apiExtensionListSrc.Items[0].Spec.UIExtensions[0].Label,
					FileName:    apiExtensionListSrc.Items[0].Spec.UIExtensions[0].FileName,
					AppName:     apiExtensionListSrc.Items[0].Spec.UIExtensions[0].AppName,
					ModuleName:  apiExtensionListSrc.Items[0].Spec.UIExtensions[0].ModuleName,
				},
			},
			AgentClusterLabels: map[string]string{},
		},
		Status: deploymentv1beta1.APIExtensionStatus{
			TokenSecretRef: deploymentv1beta1.TokenSecretRef{
				Name:           apiExtensionListSrc.Items[0].Status.TokenSecretRef.Name,
				GeneratedToken: apiExtensionListSrc.Items[0].Status.TokenSecretRef.GeneratedToken,
			},
		},
	}

	return instance
}
