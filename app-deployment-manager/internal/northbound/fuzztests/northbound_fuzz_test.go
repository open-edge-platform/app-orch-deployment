// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fuzztests

import (
	"context"
	"encoding/json"

	"os"
	"strings"

	"net/http"
	"net/http/httptest"

	"google.golang.org/grpc/metadata"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"github.com/bufbuild/protovalidate-go"
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound/mocks"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/lasso/pkg/client"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	mockerymock "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient/mockery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"testing"
)

const (
	apiVersion       = "v1.1"
	VALID_UID        = "123456-123456-123456"
	INVALID_UID      = "023456-023456-023456"
	VALID_UID_DC     = "dc-789789-789789-789789"
	KIND             = "deployments"
	KIND_DC          = "deploymentclusters"
	KIND_C           = "clusters"
	CLUSTER_NAME     = "test-cluster"
	VALID_PROJECT_ID = "0000-1111-2222-3333-4444"
)

type FuzzTestSuite struct {
	deploymentServer         *northbound.DeploymentSvc
	deploymentListSrc        deploymentv1beta1.DeploymentList
	deploymentClusterListSrc deploymentv1beta1.DeploymentClusterList
	clusterListSrc           deploymentv1beta1.ClusterList
	k8sClient                *mocks.FakeDeploymentV1
	vaultAuthMock            *mocks.VaultAuth
	opaMock                  *openpolicyagent.MockClientWithResponsesInterface
	deployInstance           *deploymentv1beta1.Deployment
	deployInstanceResp       *deploymentpb.Deployment
	protoValidator           *protovalidate.Validator
	catalogClient            *mockerymock.MockeryCatalogClient
	kc                       *kubernetes.Clientset
	matchingLabelList        []string
	ctx                      context.Context
}

type TestReporter interface {
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

func setupFuzzTest() *FuzzTestSuite {
	s := &FuzzTestSuite{}
	var st *testing.T
	var t TestReporter
	s.k8sClient = &mocks.FakeDeploymentV1{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": "test-namespacename"}}`))
		assert.NoError(st, err)
	}))

	defer ts.Close()

	s.kc = mockK8Client(ts.URL)

	// protovalidate Validator
	s.protoValidator, _ = protovalidate.New()

	// M2M auth client mock
	s.vaultAuthMock = &mocks.VaultAuth{}

	mockController := gomock.NewController(t)
	result := openpolicyagent.OpaResponse_Result{}
	_ = result.FromOpaResponseResult1(true)
	s.opaMock = openpolicyagent.NewMockClientWithResponsesInterface(mockController)
	s.opaMock.EXPECT().PostV1DataPackageRuleWithBodyWithResponse(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any()).Return(
		&openpolicyagent.PostV1DataPackageRuleResponse{
			JSON200: &openpolicyagent.OpaResponse{
				Result: result,
			},
		}, nil,
	).AnyTimes()

	s.catalogClient = mockerymock.FuzzNewMockeryCatalogClient(st)

	s.deploymentServer = northbound.NewDeployment(s.k8sClient, s.opaMock, s.kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	md := metadata.Pairs("activeprojectid", VALID_PROJECT_ID, "authorization", "test-token")
	s.ctx = metadata.NewIncomingContext(context.Background(), md)
	s.vaultAuthMock.On("GetM2MToken", s.ctx).Return("test-m2m-token", nil)

	// populates a mock deployment object
	setDeploymentListObjects(&s.deploymentListSrc)
	setDeploymentClusterListObject(&s.deploymentClusterListSrc)
	setClusterListObject(&s.clusterListSrc)

	s.deployInstanceResp = getDeployInstance(&s.deploymentListSrc)

	return s
}

func FuzzCreateDeployment(f *testing.F) {
	s := setupFuzzTest()
	os.Setenv("SECRET_SERVICE_ENABLED", "false")

	s.deployInstance = setDeployInstance(&s.deploymentListSrc, "create")
	s.deployInstanceResp = getDeployInstance(&s.deploymentListSrc)

	seedReq := &deploymentpb.CreateDeploymentRequest{
		Deployment: s.deployInstanceResp,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.CreateDeploymentRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		appDeploymentInstance := createAppDeployment(req)

		var AnyFuzzGetDpReq = &catalog.GetDeploymentPackageRequest{
			DeploymentPackageName: req.Deployment.AppName,
			Version:               req.Deployment.AppVersion,
		}

		var FuzzDpRespGood = &catalog.GetDeploymentPackageResponse{
			DeploymentPackage: &catalog.DeploymentPackage{
				Name:                  req.Deployment.AppName,
				Version:               req.Deployment.AppVersion,
				ApplicationReferences: []*catalog.ApplicationReference{},
				Profiles: []*catalog.DeploymentProfile{
					{
						Name: req.Deployment.ProfileName,
						ApplicationProfiles: map[string]string{
							req.Deployment.AppName: req.Deployment.ProfileName,
						},
					},
				},
				DefaultProfileName:      req.Deployment.ProfileName,
				ApplicationDependencies: []*catalog.ApplicationDependency{},
				Extensions:              []*catalog.APIExtension{},
				Artifacts:               []*catalog.ArtifactReference{},
				DefaultNamespaces:       map[string]string{},
			},
		}

		s.k8sClient.On(
			"Create", mocks.AnyContext, appDeploymentInstance, mock.AnythingOfType("v1.CreateOptions"),
		).Return(appDeploymentInstance, nil)

		s.k8sClient.On(
			"List", mocks.AnyContext, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		s.catalogClient.On("GetDeploymentPackage", mocks.AnyContext, AnyFuzzGetDpReq).Return(FuzzDpRespGood, nil)

		resp, err := deploymentServer.CreateDeployment(s.ctx, req)

		if err != nil {
			if !strings.Contains(err.Error(), "rpc error: code = InvalidArgument desc = validation error") {
				t.Log(err)
				assert.Nil(t, resp)
			}
		}
	})
}

func FuzzListDeployments(f *testing.F) {
	s := setupFuzzTest()

	seedReq := &deploymentpb.ListDeploymentsRequest{
		Labels: s.matchingLabelList,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.ListDeploymentsRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		resp, err := deploymentServer.ListDeployments(s.ctx, req)
		if err != nil {
			if err.Error() != `rpc error: code = InvalidArgument desc = invalid filter request` &&
				err.Error() != `rpc error: code = InvalidArgument desc = invalid order direction; must be 'asc' or 'desc'` &&
				err.Error() != `rpc error: code = InvalidArgument desc = invalid format for order by parameter` &&
				!strings.Contains(err.Error(), "rpc error: code = InvalidArgument desc = validation error") {
				assert.NoError(t, err)
				t.Log(err)
				assert.Nil(t, resp)
			}
		}
	})
}

func FuzzListDeploymentClusters(f *testing.F) {
	s := setupFuzzTest()

	seedReq := &deploymentpb.ListDeploymentClustersRequest{
		DeplId: VALID_UID,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.ListDeploymentClustersRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		resp, err := deploymentServer.ListDeploymentClusters(s.ctx, req)
		if err != nil {
			if err.Error() != `rpc error: code = InvalidArgument desc = incomplete request` &&
				err.Error() != "rpc error: code = InvalidArgument desc = validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 100 [int32.gte_lte]" &&
				!strings.Contains(err.Error(), "invalid filter request") &&
				!strings.Contains(err.Error(), "invalid order direction") &&
				!strings.Contains(err.Error(), "invalid format for order by parameter") &&
				!strings.Contains(err.Error(), "rpc error: code = NotFound desc = deployment id ") &&
				!strings.Contains(err.Error(), "value does not match regex pattern") {
				assert.NoError(t, err)
				t.Log(err, req)
				assert.Nil(t, resp)
			}
		}
	})
}

func FuzzListClusters(f *testing.F) {
	s := setupFuzzTest()

	seedReq := &deploymentpb.ListClustersRequest{
		Labels: s.matchingLabelList,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.ListClustersRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.ClusterList{}, nil).Once()

		resp, err := deploymentServer.ListClusters(s.ctx, req)

		if err != nil {
			if err.Error() != `rpc error: code = InvalidArgument desc = invalid filter request` &&
				err.Error() != `rpc error: code = InvalidArgument desc = invalid order direction; must be 'asc' or 'desc'` &&
				err.Error() != `rpc error: code = InvalidArgument desc = invalid format for order by parameter` &&
				err.Error() != "rpc error: code = InvalidArgument desc = validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 100 [int32.gte_lte]" {
				assert.NoError(t, err)
				t.Log(err)
				assert.Nil(t, resp)
			}
		}
	})
}

func FuzzDeploymentsStatus(f *testing.F) {
	s := setupFuzzTest()
	seedReq := &deploymentpb.GetDeploymentsStatusRequest{
		Labels: s.matchingLabelList,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, s.catalogClient, s.protoValidator, s.vaultAuthMock)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.GetDeploymentsStatusRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentClusterList{}, nil).Once()

		resp, err := deploymentServer.GetDeploymentsStatus(s.ctx, req)
		if err != nil {
			if !strings.Contains(err.Error(), "rpc error: code = InvalidArgument desc = validation error") {
				assert.NoError(t, err)
				t.Log(err)
				assert.Nil(t, resp)
			}
		}
	})
}

func FuzzGetDeployment(f *testing.F) {
	s := setupFuzzTest()
	seedReq1 := &deploymentpb.GetDeploymentRequest{
		DeplId: VALID_UID,
	}
	seedReq2 := &deploymentpb.GetDeploymentRequest{
		DeplId: INVALID_UID,
	}

	var seedData1 []byte
	err := proto.Unmarshal(seedData1, seedReq1)
	assert.NoError(f, err)

	var seedData2 []byte
	err = proto.Unmarshal(seedData2, seedReq2)
	assert.NoError(f, err)
	f.Add(seedData1)
	f.Add(seedData2)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, nil, nil, nil, s.protoValidator, nil)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.GetDeploymentRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		resp, err := deploymentServer.GetDeployment(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzGetCluster(f *testing.F) {
	s := setupFuzzTest()
	seedReq1 := &deploymentpb.GetClusterRequest{
		ClusterId: CLUSTER_NAME,
	}
	var seedData1 []byte
	err := proto.Unmarshal(seedData1, seedReq1)
	assert.NoError(f, err)
	f.Add(seedData1)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, nil, nil, nil, s.protoValidator, nil)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.GetClusterRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentClusterList{}, nil).Once()

		resp, err := deploymentServer.GetCluster(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzGetAppNamespace(f *testing.F) {
	s := setupFuzzTest()
	seedReq1 := &deploymentpb.GetAppNamespaceRequest{
		AppId: VALID_UID,
	}
	seedReq2 := &deploymentpb.GetAppNamespaceRequest{
		AppId: INVALID_UID,
	}

	var seedData1 []byte
	err := proto.Unmarshal(seedData1, seedReq1)
	assert.NoError(f, err)

	var seedData2 []byte
	err = proto.Unmarshal(seedData2, seedReq2)
	assert.NoError(f, err)
	f.Add(seedData1)
	f.Add(seedData2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	bundleClient, err := newMockBundleClient(ts.URL)
	assert.NoError(f, err)

	deploymentServer := northbound.NewDeployment(nil, s.opaMock, nil, bundleClient, nil, s.protoValidator, nil)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.GetAppNamespaceRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		resp, err := deploymentServer.GetAppNamespace(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzGetKubeConfig(f *testing.F) {
	s := setupFuzzTest()
	seedReq1 := &deploymentpb.GetKubeConfigRequest{
		ClusterId: CLUSTER_NAME,
	}
	var seedData1 []byte
	err := proto.Unmarshal(seedData1, seedReq1)
	assert.NoError(f, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, nil, s.protoValidator, nil)

	f.Add(seedData1)
	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.GetKubeConfigRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"Get", s.ctx, CLUSTER_NAME, mock.AnythingOfType("v1.GetOptions"),
		).Return(&s.clusterListSrc.Items[0], nil)

		s.k8sClient.On(
			"Get", s.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("v1.GetOptions"),
		).Return((*deploymentv1beta1.Cluster)(nil), k8serrors.NewNotFound(schema.GroupResource{}, ""))

		resp, err := deploymentServer.GetKubeConfig(s.ctx, req)
		if err != nil {
			t.Log(err, req)
			assert.Nil(t, resp)
		}
	})
}

func FuzzDeleteDeployment(f *testing.F) {
	s := setupFuzzTest()
	seedReq1 := &deploymentpb.DeleteDeploymentRequest{
		DeplId: VALID_UID,
	}
	seedReq2 := &deploymentpb.DeleteDeploymentRequest{
		DeplId: INVALID_UID,
	}

	var seedData1 []byte
	err := proto.Unmarshal(seedData1, seedReq1)
	assert.NoError(f, err)

	var seedData2 []byte
	err = proto.Unmarshal(seedData2, seedReq2)
	assert.NoError(f, err)

	f.Add(seedData1)
	f.Add(seedData2)

	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.DeleteDeploymentRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil)

		resp, err := s.deploymentServer.DeleteDeployment(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzUpdateDeployment(f *testing.F) {
	s := setupFuzzTest()

	s.deploymentListSrc.Items[0].Spec.DeploymentPackageRef.Version = "0.1.1"

	s.deployInstanceResp.AppVersion = "0.1.1"

	seedReq := &deploymentpb.UpdateDeploymentRequest{
		DeplId:     VALID_UID,
		Deployment: s.deployInstanceResp,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)
	f.Add(seedData)
	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.UpdateDeploymentRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		s.k8sClient.On(
			"Update", context.Background(), mock.AnythingOfType("string"), s.deployInstance, mock.AnythingOfType("v1.UpdateOptions"),
		).Return(s.deployInstance, nil)

		resp, err := s.deploymentServer.UpdateDeployment(s.ctx, req)
		if err != nil {
			t.Log(err)
			assert.Nil(t, resp)
		}
	})
}

func FuzzListDeploymentsPerCluster(f *testing.F) {
	s := setupFuzzTest()
	seedReq := &deploymentpb.ListDeploymentsPerClusterRequest{
		ClusterId: CLUSTER_NAME,
	}

	var seedData []byte
	err := proto.Unmarshal(seedData, seedReq)
	assert.NoError(f, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
	}))

	defer ts.Close()

	kc := mockK8Client(ts.URL)

	deploymentServer := northbound.NewDeployment(s.k8sClient, s.opaMock, kc, nil, nil, s.protoValidator, nil)

	f.Add(seedData)
	f.Fuzz(func(t *testing.T, seedData []byte) {
		consumer := fuzz.NewConsumer(seedData)
		req := &deploymentpb.ListDeploymentsPerClusterRequest{}
		err = consumer.GenerateStruct(&req)
		if err != nil {
			return
		}

		req.PageSize = 1

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentList{}, nil).Once()

		s.k8sClient.On(
			"List", s.ctx, mock.AnythingOfType("v1.ListOptions"),
		).Return(&deploymentv1beta1.DeploymentClusterList{}, nil).Once()

		resp, err := deploymentServer.ListDeploymentsPerCluster(s.ctx, req)
		if err != nil {
			if !strings.Contains(err.Error(), "value length must be at least 1 characters [string.min_len]") &&
				!strings.Contains(err.Error(), "value must be greater than or equal to 0") &&
				!strings.Contains(err.Error(), "value does not match regex pattern") &&
				!strings.Contains(err.Error(), "failed to retrieve deployment with deployment UID in deploymentcluster CR UID") {
				assert.NoError(t, err)
				t.Log(err)
				assert.Nil(t, resp)
			}
		}
	})
}

func createAppDeployment(req *deploymentpb.CreateDeploymentRequest) *deploymentv1beta1.Deployment {
	d := req.Deployment

	labelList := map[string]string{
		"app.kubernetes.io/name":                     "deployment",
		"app.kubernetes.io/instance":                 "test-deployment",
		"app.kubernetes.io/part-of":                  "app-deployment-manager",
		"app.kubernetes.io/managed-by":               "kustomize",
		"app.kubernetes.io/created-by":               "app-deployment-manager",
		"app.edge-orchestrator.intel.com/project-id": VALID_PROJECT_ID,
	}

	dpRef := deploymentv1beta1.DeploymentPackageRef{
		Name:        d.AppName,
		Version:     d.AppVersion,
		ProfileName: d.ProfileName,
	}

	networkRef := corev1.ObjectReference{
		Name:       "network-1",
		Kind:       "Network",
		APIVersion: "network.edge-orchestrator.intel/v1",
	}

	instance := &deploymentv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: VALID_PROJECT_ID,
			Labels:    labelList,
			UID:       types.UID(VALID_UID),
		},
		Spec: deploymentv1beta1.DeploymentSpec{
			DisplayName:          d.DisplayName,
			Project:              "app.edge-orchestrator.intel.com",
			DeploymentPackageRef: dpRef,
			Applications:         []deploymentv1beta1.Application{},
			DeploymentType:       "auto-scaling",
			NetworkRef:           networkRef,
			ChildDeploymentList:  make(map[string]deploymentv1beta1.DependentDeploymentRef),
		},
	}

	return instance
}

func setDeploymentListObjects(deploymentListSrc *deploymentv1beta1.DeploymentList) {
	deploymentListSrc.TypeMeta.Kind = "deployments"
	deploymentListSrc.TypeMeta.APIVersion = apiVersion

	deploymentListSrc.ListMeta.ResourceVersion = "6"
	deploymentListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	deploymentListSrc.ListMeta.RemainingItemCount = &remainingItem

	deploymentListSrc.Items = make([]deploymentv1beta1.Deployment, 3)
	setDeploymentObject(&deploymentListSrc.Items[0])
	setDeploymentObject(&deploymentListSrc.Items[1])
	setDeploymentObject(&deploymentListSrc.Items[2])
}

func setDeploymentClusterListObject(deploymentClusterListSrc *deploymentv1beta1.DeploymentClusterList) {
	deploymentClusterListSrc.TypeMeta.Kind = KIND_DC
	deploymentClusterListSrc.TypeMeta.APIVersion = apiVersion

	deploymentClusterListSrc.ListMeta.ResourceVersion = "6"
	deploymentClusterListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	deploymentClusterListSrc.ListMeta.RemainingItemCount = &remainingItem

	deploymentClusterListSrc.Items = make([]deploymentv1beta1.DeploymentCluster, 3)
	setDeploymentClusterObject(&deploymentClusterListSrc.Items[0])
}

func setDeploymentClusterObject(deploymentClusterSrc *deploymentv1beta1.DeploymentCluster) {
	deploymentClusterSrc.ObjectMeta.Name = "test-deployment-cluster"
	deploymentClusterSrc.ObjectMeta.GenerateName = "test-generate-name"
	deploymentClusterSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	deploymentClusterSrc.ObjectMeta.UID = types.UID(VALID_UID_DC)
	deploymentClusterSrc.ObjectMeta.ResourceVersion = "6"
	deploymentClusterSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	deploymentClusterSrc.ObjectMeta.CreationTimestamp = currentTime
	deploymentClusterSrc.ObjectMeta.DeletionTimestamp = &currentTime

	deploymentClusterSrc.ObjectMeta.Labels = make(map[string]string)
	deploymentClusterSrc.ObjectMeta.Labels["app.kubernetes.io/name"] = "deployment-cluster"
	deploymentClusterSrc.ObjectMeta.Labels["app.kubernetes.io/instance"] = deploymentClusterSrc.ObjectMeta.Name
	deploymentClusterSrc.ObjectMeta.Labels["app.kubernetes.io/part-of"] = "app-deployment-manager"
	deploymentClusterSrc.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "kustomize"
	deploymentClusterSrc.ObjectMeta.Labels["app.kubernetes.io/created-by"] = "app-deployment-manager"
	deploymentClusterSrc.ObjectMeta.Labels[string(deploymentv1beta1.DeploymentID)] = string(types.UID(VALID_UID))
	deploymentClusterSrc.ObjectMeta.Labels[string(deploymentv1beta1.ClusterName)] = "test-cluster-id"

	deploymentClusterSrc.Status.Name = "test-cluster-displayname"
	status := deploymentv1beta1.Status{
		State:   "Running",
		Message: "",
		Summary: deploymentv1beta1.Summary{
			Total:   1,
			Running: 2,
			Down:    1,
			Unknown: 0,
		},
	}

	deploymentClusterSrc.Status.Apps = make([]deploymentv1beta1.App, 2)

	deploymentClusterSrc.Status.Apps[0] = deploymentv1beta1.App{
		Name: "test0",
		Id:   "0.1.0",
		// DeploymentGeneration:          "default",
		Status: status,
	}

	deploymentClusterSrc.Status.Apps[1] = deploymentv1beta1.App{
		Name: "test1",
		Id:   "0.1.0",
		// DeploymentGeneration:          "default",
		Status: status,
	}

	deploymentClusterSrc.Status.Status.State = "Running"
	deploymentClusterSrc.Status.Status.Message = ""
	deploymentClusterSrc.Status.Status.Summary.Total = 1
	deploymentClusterSrc.Status.Status.Summary.Running = 2
	deploymentClusterSrc.Status.Status.Summary.Down = 1
	deploymentClusterSrc.Status.Status.Summary.Unknown = 0
}

func setDeploymentObject(deploymentSrc *deploymentv1beta1.Deployment) {
	deploymentSrc.ObjectMeta.Name = "test-deployment"
	deploymentSrc.ObjectMeta.GenerateName = "test-generate-name"
	deploymentSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	deploymentSrc.ObjectMeta.UID = types.UID(VALID_UID)
	deploymentSrc.ObjectMeta.ResourceVersion = "6"
	deploymentSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	deploymentSrc.ObjectMeta.CreationTimestamp = currentTime
	deploymentSrc.ObjectMeta.DeletionTimestamp = &currentTime

	deploymentSrc.ObjectMeta.Labels = make(map[string]string)
	deploymentSrc.ObjectMeta.Labels["app.kubernetes.io/name"] = "deployment"
	deploymentSrc.ObjectMeta.Labels["app.kubernetes.io/instance"] = deploymentSrc.ObjectMeta.Name
	deploymentSrc.ObjectMeta.Labels["app.kubernetes.io/part-of"] = "app-deployment-manager"
	deploymentSrc.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "kustomize"
	deploymentSrc.ObjectMeta.Labels["app.kubernetes.io/created-by"] = "app-deployment-manager"

	deploymentSrc.Spec.DisplayName = "test display name"
	deploymentSrc.Spec.Project = "app.edge-orchestrator.intel.com"
	deploymentSrc.Spec.DeploymentPackageRef.Name = "wordpress"
	deploymentSrc.Spec.DeploymentPackageRef.Version = "0.1.0"
	deploymentSrc.Spec.DeploymentPackageRef.ProfileName = "default"
	deploymentSrc.Spec.DeploymentType = deploymentv1beta1.AutoScaling

	deploymentSrc.Spec.Applications = make([]deploymentv1beta1.Application, 1)
	deploymentSrc.Spec.Applications[0] = deploymentv1beta1.Application{
		Name:                "wordpress",
		NamespaceLabels:     map[string]string{},
		ProfileSecretName:   "ProfileSecretName",
		ValueSecretName:     "ValueSecretName",
		DependsOn:           []string{"dependency"},
		RedeployAfterUpdate: false,
		// IgnoreResources:          "test-ignoreResources",
		HelmApp: &deploymentv1beta1.HelmApp{
			Chart:   "wordpress",
			Version: "15.2.42",
			Repo:    "https://charts.bitnami.com/bitnami",
		},
	}

	var label []map[string]string
	deploymentSrc.Spec.Applications[0].Targets = label
	if deploymentSrc.Spec.DeploymentType == deploymentv1beta1.Targeted {
		deploymentSrc.Spec.Applications[0].Targets = []map[string]string{{"test-ClusterId-key": "test-ClusterId"}}
	} else {
		deploymentSrc.Spec.Applications[0].Targets = []map[string]string{{"test": "foo"}}
	}

	deploymentSrc.Status.Summary.Total = 3
	deploymentSrc.Status.Summary.Running = 2
	deploymentSrc.Status.Summary.Down = 1
	deploymentSrc.Status.Summary.Unknown = 0
}

func setClusterListObject(clusterListSrc *deploymentv1beta1.ClusterList) {
	clusterListSrc.TypeMeta.Kind = KIND_C
	clusterListSrc.TypeMeta.APIVersion = apiVersion

	clusterListSrc.ListMeta.ResourceVersion = "6"
	clusterListSrc.ListMeta.Continue = "yes"
	remainingItem := int64(10)
	clusterListSrc.ListMeta.RemainingItemCount = &remainingItem

	clusterListSrc.Items = make([]deploymentv1beta1.Cluster, 3)
	setClusterObject(&clusterListSrc.Items[0])
}

func setClusterObject(clusterSrc *deploymentv1beta1.Cluster) {
	clusterSrc.ObjectMeta.Name = CLUSTER_NAME
	clusterSrc.ObjectMeta.GenerateName = "test-generate-name"
	clusterSrc.ObjectMeta.Namespace = VALID_PROJECT_ID
	clusterSrc.ObjectMeta.UID = types.UID(VALID_UID)
	clusterSrc.ObjectMeta.ResourceVersion = "6"
	clusterSrc.ObjectMeta.Generation = 24456

	currentTime := metav1.Now()
	clusterSrc.ObjectMeta.CreationTimestamp = currentTime
	clusterSrc.ObjectMeta.DeletionTimestamp = &currentTime

	clusterSrc.ObjectMeta.Labels = make(map[string]string)
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/name"] = "custer"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/instance"] = clusterSrc.ObjectMeta.Name
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/part-of"] = "app-deployment-manager"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/managed-by"] = "kustomize"
	clusterSrc.ObjectMeta.Labels["app.kubernetes.io/created-by"] = "app-deployment-manager"

	clusterSrc.Spec.DisplayName = "test display name"
	clusterSrc.Spec.Name = "test name"
	clusterSrc.Spec.KubeConfigSecretName = "kubeconfig secret name"
}

func getDeployInstance(deploymentListSrc *deploymentv1beta1.DeploymentList) *deploymentpb.Deployment {
	OverrideValuesList := make([]*deploymentpb.OverrideValues, 1)
	var valuesStrPb *structpb.Struct

	rawMsg := json.RawMessage("{\"test\":\"foo\"}")
	_ = json.Unmarshal(rawMsg, &valuesStrPb)
	OverrideValuesList[0] = &deploymentpb.OverrideValues{
		AppName:         "test-appname",
		TargetNamespace: "apps",
		Values:          valuesStrPb,
	}

	TargetClustersList := make([]*deploymentpb.TargetClusters, 1)
	TargetClustersList[0] = &deploymentpb.TargetClusters{
		AppName: "test-appname",
		Labels:  map[string]string{"test": "foo"},
	}

	// Create CR Deployment summary
	summary := &deploymentpb.Summary{
		Total:   int32(deploymentListSrc.Items[0].Status.Summary.Total),
		Running: int32(deploymentListSrc.Items[0].Status.Summary.Running),
		Down:    int32(deploymentListSrc.Items[0].Status.Summary.Down),
		Unknown: int32(deploymentListSrc.Items[0].Status.Summary.Unknown),
		Type:    "test-type",
	}

	// Create CR Deployment status
	status := &deploymentpb.Deployment_Status{
		State:   deploymentpb.State_RUNNING,
		Message: "test",
		Summary: summary,
	}

	deployInstanceResp := &deploymentpb.Deployment{
		Name:           deploymentListSrc.Items[0].ObjectMeta.Name,
		AppName:        deploymentListSrc.Items[0].Spec.DeploymentPackageRef.Name,
		DisplayName:    deploymentListSrc.Items[0].Spec.DisplayName,
		AppVersion:     deploymentListSrc.Items[0].Spec.DeploymentPackageRef.Version,
		ProfileName:    deploymentListSrc.Items[0].Spec.DeploymentPackageRef.ProfileName,
		Status:         status,
		OverrideValues: OverrideValuesList,
		TargetClusters: TargetClustersList,
	}
	return deployInstanceResp
}

func setDeployInstance(deploymentListSrc *deploymentv1beta1.DeploymentList, scenario string) *deploymentv1beta1.Deployment {
	if scenario == "create" {
		app := make([]deploymentv1beta1.Application, 1)
		app[0] = deploymentv1beta1.Application{
			Name:                "wordpress",
			Version:             "0.1.0",
			NamespaceLabels:     map[string]string{},
			Targets:             make([]map[string]string, 0),
			DependsOn:           []string{"dependency"},
			RedeployAfterUpdate: false,
			HelmApp: &deploymentv1beta1.HelmApp{
				Chart:         "wordpress",
				Version:       "15.2.42",
				Repo:          "https://charts.bitnami.com/bitnami",
				ImageRegistry: "https://charts.bitnami.com/bitnami",
			},
			DependentDeploymentPackages: make(map[string]deploymentv1beta1.DeploymentPackageRef),
		}

		dpRef := deploymentv1beta1.DeploymentPackageRef{
			Name:        "wordpress",
			Version:     "0.1.0",
			ProfileName: "default",
		}

		instance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentListSrc.Items[0].ObjectMeta.Name,
				Namespace: deploymentListSrc.Items[0].ObjectMeta.Namespace,
				Labels:    deploymentListSrc.Items[0].ObjectMeta.Labels,
				UID:       types.UID(VALID_UID),
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName:          deploymentListSrc.Items[0].Spec.DisplayName,
				Project:              deploymentListSrc.Items[0].Spec.Project,
				DeploymentPackageRef: dpRef,
				Applications:         app,
				DeploymentType:       deploymentListSrc.Items[0].Spec.DeploymentType,
				ChildDeploymentList:  make(map[string]deploymentv1beta1.DependentDeploymentRef),
			},
		}
		return instance
	} else {
		app := make([]deploymentv1beta1.Application, 1)
		app[0] = deploymentv1beta1.Application{
			Name:                "wordpress",
			Version:             "0.1.0",
			NamespaceLabels:     map[string]string{},
			Targets:             make([]map[string]string, 0),
			DependsOn:           []string{"dependency"},
			RedeployAfterUpdate: false,
			HelmApp: &deploymentv1beta1.HelmApp{
				Chart:         "wordpress",
				Version:       "15.2.42",
				Repo:          "https://charts.bitnami.com/bitnami",
				ImageRegistry: "https://charts.bitnami.com/bitnami",
			},
			DependentDeploymentPackages: make(map[string]deploymentv1beta1.DeploymentPackageRef),
		}

		dpRef := deploymentv1beta1.DeploymentPackageRef{
			Name:        "wordpress",
			Version:     "0.1.0",
			ProfileName: "default",
		}

		instance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            deploymentListSrc.Items[0].ObjectMeta.Name,
				Namespace:       deploymentListSrc.Items[0].ObjectMeta.Namespace,
				Labels:          deploymentListSrc.Items[0].ObjectMeta.Labels,
				ResourceVersion: deploymentListSrc.Items[0].ObjectMeta.ResourceVersion,
				UID:             types.UID(VALID_UID),
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName:          deploymentListSrc.Items[0].Spec.DisplayName,
				Project:              deploymentListSrc.Items[0].Spec.Project,
				DeploymentPackageRef: dpRef,
				Applications:         app,
				DeploymentType:       deploymentListSrc.Items[0].Spec.DeploymentType,
			},
		}
		return instance
	}
}

func mockK8Client(tsUrl string) *kubernetes.Clientset {
	config := &rest.Config{
		Host: tsUrl,
	}

	gv := metav1.SchemeGroupVersion
	config.GroupVersion = &gv

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentType = "application/json"

	_kClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil
	}

	return _kClient
}

func newMockBundleClient(tsUrl string) (*fleet.BundleClient, error) {
	bundleClient := &fleet.BundleClient{}

	newConfig := rest.Config{
		Host: tsUrl,
	}
	newConfig.ContentConfig.GroupVersion = &schema.GroupVersion{}
	newConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	newConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.RESTClientFor(&newConfig)
	if err != nil {
		return nil, err
	}
	cl := client.NewClient(schema.GroupVersionResource{
		Group:    fleetv1alpha1.SchemeGroupVersion.Group,
		Version:  fleetv1alpha1.SchemeGroupVersion.Version,
		Resource: "bundles",
	}, "Bundle", true, restClient, 0)
	if err != nil {
		return nil, err
	}
	bundleClient.Client = cl
	return bundleClient, nil
}
