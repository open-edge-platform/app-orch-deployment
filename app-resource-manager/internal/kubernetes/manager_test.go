// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"os"
	"testing"
	"time"

	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	admmock "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm/mocks"
	kubernetesmocks "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubernetes/mocks"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynFake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testAppID                      = "appID1"
	testClusterID                  = "cluster1"
	testReplicaSet                 = "test-replicaset"
	testStatefulSet                = "test-statefulset"
	testDaemoSet                   = "test-daemonset"
	testNamespace                  = "app-orchs"
	testServiceName                = "test-service"
	testPodNameReplicaSet          = "test-pod-replicaset"
	testPodNameStatefulSet         = "test-pod-statefulset"
	testPodNameDaemonSet           = "test-pod-daemonset"
	testIngressNameHostOnly        = "test-ingress-host-only"
	testIngressNameAnnotationOnly  = "test-ingress-annotation-only"
	testFQDNAnnotationOnly         = "test-service.test-domain.com"
	testServiceProxyAnnotationOnly = "https:8080,80"
	testFQDNHostOnly               = "test-service-host-only.test-domain.com"
	testKubeConfig                 = "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: \n    server: https://127.0.0.1:39623\n  name: kind-kind\ncontexts:\n- context:\n    cluster: kind-kind\n    user: kind-kind\n  name: kind-kind\ncurrent-context: kind-kind\nkind: Config\npreferences: {}\nusers:\n- name: kind-kind\n  user:\n    client-certificate-data: "
)

const (
	testConfigValue = `
 appDeploymentManager:
    endpoint: "http://adm-api.orch-app.svc:8081"
 webSocketServer:
    protocol: "wss"
    hostName: "vnc.kind.internal"
    sessionLimitPerIP: 0
    sessionLimitPerAccount: 0
    readLimitByte: 0
    dlIdleTimeoutMin: 0
    ulIdleTimeoutMin: 0
    allowedOrigins:
      - https://vnc.kind.internal`
)

func newDefinedHostOnlyIngress() *v1.Ingress {
	return &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressNameHostOnly,
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation:           testFQDNAnnotationOnly,
				AnnotationKeyForAppID:           testAppID,
				IngressHostnameSourceAnnotation: DefinedHostOnly,
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: testFQDNHostOnly,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: testServiceName,
											Port: v1.ServiceBackendPort{
												Name:   "http",
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newServiceEndpoint() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServiceName,
			Namespace: testNamespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
			},
		},
	}
}

func newPodReplicaSet(rs *appsv1.ReplicaSet) *corev1.Pod {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPodNameReplicaSet,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       rs.ObjectMeta.Name,
					UID:        rs.ObjectMeta.UID,
					Controller: &controller,
				},
			},
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation: testFQDNAnnotationOnly,
				AnnotationKeyForAppID: testAppID,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "testContainer1",
					Image: "testImage1",
				},
				{
					Name:  "testContainer2",
					Image: "testImage2",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "testContainer1",
					Ready: true,
					Image: "testImage1",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
				{
					Name:  "testContainer2",
					Ready: true,
					Image: "testImage2",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
			},
		},
	}
}

func newPodStatefulSet(sfs *appsv1.StatefulSet) *corev1.Pod {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPodNameStatefulSet,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Name:       sfs.ObjectMeta.Name,
					UID:        sfs.ObjectMeta.UID,
					Controller: &controller,
				},
			},
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation: testFQDNAnnotationOnly,
				AnnotationKeyForAppID: testAppID,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "testContainer1",
					Image: "testImage1",
				},
				{
					Name:  "testContainer2",
					Image: "testImage2",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "testContainer1",
					Ready: true,
					Image: "testImage1",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
				{
					Name:  "testContainer2",
					Ready: true,
					Image: "testImage2",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
			},
		},
	}
}

func newPodDaemonSet(ds *appsv1.DaemonSet) *corev1.Pod {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPodNameDaemonSet,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
					Name:       ds.ObjectMeta.Name,
					UID:        ds.ObjectMeta.UID,
					Controller: &controller,
				},
			},
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation: testFQDNAnnotationOnly,
				AnnotationKeyForAppID: testAppID,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "testContainer1",
					Image: "testImage1",
				},
				{
					Name:  "testContainer2",
					Image: "testImage2",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "testContainer1",
					Ready: true,
					Image: "testImage1",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
				{
					Name:  "testContainer2",
					Ready: true,
					Image: "testImage2",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(time.Now()),
						},
					},
				},
			},
		},
	}
}

func newServiceAnnotationOnly() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServiceName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation:      testFQDNAnnotationOnly,
				ServiceProxyPortAnnotation: testServiceProxyAnnotationOnly,
				AnnotationKeyForAppID:      testAppID,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "https",
					Port:     8080,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "sctp",
					Port:     5000,
					Protocol: corev1.ProtocolSCTP,
				},
			},
		},
	}
}

func newAnnotaionOnlyIngress() *v1.Ingress {
	return &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIngressNameAnnotationOnly,
			Namespace: testNamespace,
			Annotations: map[string]string{
				ExternalDNSAnnotation:           testFQDNAnnotationOnly,
				AnnotationKeyForAppID:           testAppID,
				IngressHostnameSourceAnnotation: AnnotationOnly,
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: testFQDNHostOnly,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: testServiceName,
											Port: v1.ServiceBackendPort{
												Name:   "http",
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newReplicaSet() *appsv1.ReplicaSet {
	var replicas int32 = 1
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: testReplicaSet,
			Annotations: map[string]string{
				AnnotationKeyForAppID: testAppID,
			},
		},

		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}
}

func newStatefulSet() *appsv1.StatefulSet {
	var replicas int32 = 1
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: testReplicaSet,
			Annotations: map[string]string{
				AnnotationKeyForAppID: testAppID,
			},
		},

		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}
}

func newDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDaemoSet,
			Annotations: map[string]string{
				AnnotationKeyForAppID: testAppID,
			},
		},

		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}
}

type KubernetesManagerTestSuite struct {
	suite.Suite
	ctx                   context.Context
	cancel                context.CancelFunc
	admClientMock         *admmock.MockADMClient
	kubernetesManagerMock *kubernetesmocks.MockKubernetesManager
	kubernetesManager     Manager
	configFile            *os.File
}

func (s *KubernetesManagerTestSuite) SetupSuite() {

}

func (s *KubernetesManagerTestSuite) TearDownSuite() {
	err := os.RemoveAll(s.configFile.Name())
	assert.NoError(s.T(), err)
}

func (s *KubernetesManagerTestSuite) SetupTest() {
	s.T().Log("Setting up the Kubernetes Manager test suite")
	s.admClientMock = admmock.NewMockADMClient(s.T())
	s.kubernetesManager = NewManager("", s.admClientMock)
	s.kubernetesManagerMock = kubernetesmocks.NewMockKubernetesManager(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	configFile, err := os.CreateTemp(s.T().TempDir(), "config.yaml")

	assert.NoError(s.T(), err)
	err = os.WriteFile(configFile.Name(), []byte(testConfigValue), 0600)
	assert.NoError(s.T(), err)
	s.configFile = configFile

	err = os.Setenv("RATE_LIMITER_QPS", "25")
	assert.NoError(s.T(), err)
	os.Setenv("RATE_LIMITER_BURST", "1500")
	assert.NoError(s.T(), err)

	os.Setenv(serviceProxyDomainName, "https://app-service-proxy.kind.internal")
	assert.NoError(s.T(), err)

}

func (s *KubernetesManagerTestSuite) TestDeletePodWorkload() {
	fakeK8sClient := fake.NewSimpleClientset()
	fakeDynamicK8sClient := dynFake.NewSimpleDynamicClient(runtime.NewScheme())

	_, err := fakeK8sClient.CoreV1().Namespaces().Create(s.ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}, metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return fakeK8sClient, nil
	}
	origDynamicK8sClient := getDynamicK8sClient
	getDynamicK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (dynamic.Interface, error) {
		return fakeDynamicK8sClient, nil
	}

	replicaSetInstance := newReplicaSet()
	createRs, err := fakeK8sClient.AppsV1().ReplicaSets(testNamespace).Create(s.ctx, replicaSetInstance, metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.CoreV1().Pods(testNamespace).Create(s.ctx, newPodReplicaSet(createRs), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	s.admClientMock.On("GetAppNamespace", s.ctx, testAppID).Return(testNamespace, nil)

	podWorkloads, err := s.kubernetesManager.GetPodWorkloads(s.ctx, testAppID, testClusterID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(podWorkloads))

	err = s.kubernetesManager.DeletePod(s.ctx, testClusterID, testNamespace, testPodNameReplicaSet)
	assert.NoError(s.T(), err)

	podWorkloads, err = s.kubernetesManager.GetPodWorkloads(s.ctx, testAppID, testClusterID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, len(podWorkloads))

	getK8sClient = origGetK8sClient
	getDynamicK8sClient = origDynamicK8sClient
}

func (s *KubernetesManagerTestSuite) TestGetPodWorkloadsClientError() {
	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return nil, errors.NewInternal("internal error")
	}

	podWorkloads, err := s.kubernetesManager.GetPodWorkloads(s.ctx, testAppID, testClusterID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), podWorkloads)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestGetAppEndpointsClientError() {
	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return nil, errors.NewInternal("internal error")
	}

	appEndpoints, err := s.kubernetesManager.GetAppEndpointsV2(s.ctx, testAppID, testClusterID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), appEndpoints)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestGetAppEndpointsInvalidAppID() {
	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return nil, errors.NewInternal("internal error")
	}

	appEndpoints, err := s.kubernetesManager.GetAppEndpointsV2(s.ctx, "", testClusterID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), appEndpoints)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestDeletePodError() {
	fakeK8sClient := fake.NewSimpleClientset()

	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return fakeK8sClient, nil
	}

	err := s.kubernetesManager.DeletePod(s.ctx, "", testNamespace, testPodNameStatefulSet)
	assert.Error(s.T(), err)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestDeletePodClientError() {
	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return nil, errors.NewInternal("internal error")
	}

	err := s.kubernetesManager.DeletePod(s.ctx, testClusterID, testNamespace, testPodNameStatefulSet)
	assert.Error(s.T(), err)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestGetPodWorkloads() {
	fakeK8sClient := fake.NewSimpleClientset()
	fakeDynamicK8sClient := dynFake.NewSimpleDynamicClient(runtime.NewScheme())

	replicaSetInstance := newReplicaSet()
	createRs, err := fakeK8sClient.AppsV1().ReplicaSets(testNamespace).Create(s.ctx, replicaSetInstance, metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.CoreV1().Pods(testNamespace).Create(s.ctx, newPodReplicaSet(createRs), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	statefulSetInstance := newStatefulSet()
	createSfs, err := fakeK8sClient.AppsV1().StatefulSets(testNamespace).Create(s.ctx, statefulSetInstance, metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.CoreV1().Pods(testNamespace).Create(s.ctx, newPodStatefulSet(createSfs), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	daemonSetInstance := newDaemonSet()
	createds, err := fakeK8sClient.AppsV1().DaemonSets(testNamespace).Create(s.ctx, daemonSetInstance, metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.CoreV1().Pods(testNamespace).Create(s.ctx, newPodDaemonSet(createds), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	s.admClientMock.On("GetAppNamespace", s.ctx, testAppID).Return(testNamespace, nil)

	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return fakeK8sClient, nil
	}
	origDynamicK8sClient := getDynamicK8sClient
	getDynamicK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (dynamic.Interface, error) {
		return fakeDynamicK8sClient, nil
	}

	podWorkloads, err := s.kubernetesManager.GetPodWorkloads(s.ctx, testAppID, testClusterID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 3, len(podWorkloads))
	s.T().Log(podWorkloads)

	workLoad1 := podWorkloads[0].GetPod()
	assert.Equal(s.T(), resourceapiv2.PodStatus_STATE_RUNNING, workLoad1.Status.State)

	workLoad2 := podWorkloads[1].GetPod()
	assert.Equal(s.T(), resourceapiv2.PodStatus_STATE_RUNNING, workLoad2.Status.State)

	workLoad3 := podWorkloads[2].GetPod()
	assert.Equal(s.T(), resourceapiv2.PodStatus_STATE_RUNNING, workLoad3.Status.State)

	getK8sClient = origGetK8sClient
	getDynamicK8sClient = origDynamicK8sClient

}

func (s *KubernetesManagerTestSuite) TestGetAppEndpoints() {
	fakeK8sClient := fake.NewSimpleClientset()

	s.admClientMock.On("GetAppNamespace", s.ctx, testAppID).Return(testNamespace, nil)

	_, err := fakeK8sClient.CoreV1().Services(testNamespace).Create(s.ctx, newServiceAnnotationOnly(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.CoreV1().Endpoints(testNamespace).Create(s.ctx, newServiceEndpoint(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.NetworkingV1().Ingresses(testNamespace).Create(s.ctx, newDefinedHostOnlyIngress(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	_, err = fakeK8sClient.NetworkingV1().Ingresses(testNamespace).Create(s.ctx, newAnnotaionOnlyIngress(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)

	origGetK8sClient := getK8sClient
	getK8sClient = func(ctx context.Context, clusterID string, admClient adm.Client, configPath string) (kubernetes.Interface, error) {
		return fakeK8sClient, nil
	}

	endpointsV2, err := s.kubernetesManager.GetAppEndpointsV2(s.ctx, testAppID, testClusterID)
	assert.NoError(s.T(), err)
	s.T().Log(endpointsV2)
	assert.Len(s.T(), endpointsV2, 3)
	assert.Equal(s.T(), testFQDNAnnotationOnly, endpointsV2[0].Fqdns[0].Fqdn)
	assert.Equal(s.T(), testFQDNAnnotationOnly, endpointsV2[1].Fqdns[0].Fqdn)
	assert.Equal(s.T(), testFQDNHostOnly, endpointsV2[2].Fqdns[0].Fqdn)
	assert.Equal(s.T(), "https://app-service-proxy.kind.internal/app-service-proxy-index.html?project=nil&cluster=cluster1&namespace=app-orchs&service=test-service:80", endpointsV2[0].Ports[0].ServiceProxyUrl)
	assert.Equal(s.T(), "https://app-service-proxy.kind.internal/app-service-proxy-index.html?project=nil&cluster=cluster1&namespace=app-orchs&service=https:test-service:8080", endpointsV2[0].Ports[1].ServiceProxyUrl)

	getK8sClient = origGetK8sClient
}

func (s *KubernetesManagerTestSuite) TestGetK8sClient() {

	s.admClientMock.On("GetKubeConfig", context.Background(), testClusterID).
		Return([]byte(testKubeConfig), nil)

	k8sClient, err := getK8sClient(context.Background(), testClusterID, s.admClientMock, s.configFile.Name())
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), k8sClient)
}

func (s *KubernetesManagerTestSuite) TestGetIngressAppEndpointsDefinedHostOnly() {
	fakeK8sClient := fake.NewSimpleClientset()

	_, err := fakeK8sClient.NetworkingV1().Ingresses(testNamespace).Create(s.ctx, newDefinedHostOnlyIngress(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	endpoints, err := getIngressAppEndpointsV2(s.ctx, fakeK8sClient, testAppID, testNamespace)
	assert.NoError(s.T(), err)
	s.T().Log(endpoints)
	assert.Len(s.T(), endpoints, 1)
	assert.Equal(s.T(), testFQDNHostOnly, endpoints[0].Fqdns[0].Fqdn)
	assert.Equal(s.T(), testIngressNameHostOnly, endpoints[0].Name)

}

func (s *KubernetesManagerTestSuite) TestGetIngressAppEndpointsAnnotationOnly() {
	fakeK8sClient := fake.NewSimpleClientset()

	_, err := fakeK8sClient.NetworkingV1().Ingresses(testNamespace).Create(s.ctx, newAnnotaionOnlyIngress(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	endpoints, err := getIngressAppEndpointsV2(s.ctx, fakeK8sClient, testAppID, testNamespace)
	assert.NoError(s.T(), err)
	s.T().Log(endpoints)
	assert.Len(s.T(), endpoints, 1)
	assert.Equal(s.T(), testFQDNAnnotationOnly, endpoints[0].Fqdns[0].Fqdn)
	assert.Equal(s.T(), testIngressNameAnnotationOnly, endpoints[0].Name)

}

func (s *KubernetesManagerTestSuite) TestGetServiceAppEndpoints() {
	fakeK8sClient := fake.NewSimpleClientset()

	_, err := fakeK8sClient.CoreV1().Services(testNamespace).Create(s.ctx, newServiceAnnotationOnly(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	_, err = fakeK8sClient.CoreV1().Endpoints(testNamespace).Create(s.ctx, newServiceEndpoint(), metav1.CreateOptions{})
	assert.NoError(s.T(), err)
	endpoints, err := getServiceAppEndpointsV2(s.ctx, fakeK8sClient, "nil", testAppID, testNamespace, testClusterID)
	assert.NoError(s.T(), err)
	s.T().Log(endpoints)
	assert.Len(s.T(), endpoints, 1)
	assert.Len(s.T(), endpoints[0].Ports, 3)
	assert.Equal(s.T(), testFQDNAnnotationOnly, endpoints[0].Fqdns[0].Fqdn)
	assert.Equal(s.T(), endpoints[0].EndpointStatus.State, resourceapiv2.EndpointStatus_STATE_READY)
}

func TestKubernetesManager(t *testing.T) {
	suite.Run(t, new(KubernetesManagerTestSuite))
}
func TestNewManager(t *testing.T) {
	mgr := NewManager("", nil)
	assert.NotNil(t, mgr)
}
