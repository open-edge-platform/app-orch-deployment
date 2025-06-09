// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	resourceapiv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	ExternalDNSAnnotation           = "external-dns.alpha.kubernetes.io/hostname"
	ServiceProxyPortAnnotation      = "service-proxy.app.orchestrator.io/ports"
	AnnotationKeyForAppID           = "meta.helm.sh/release-name"
	IngressHostnameSourceAnnotation = "dns.alpha.kubernetes.io/ingress-hostname-source"
	AnnotationOnly                  = "annotation-only"
	DefinedHostOnly                 = "defined-hosts-only"
	LabelKeyForVirtHandler          = "kubevirt.io"
	serviceProxyDomainName          = "SERVICE_PROXY_DOMAIN_NAME"
)

var log = dazl.GetPackageLogger()

func NewManager(configPath string, admClient adm.Client) Manager {
	return &manager{
		configPath: configPath,
		admClient:  admClient,
	}
}

//go:generate mockery --name Manager --filename kubernetes_manager_mock.go --structname MockKubernetesManager
type Manager interface {
	GetAppEndpointsV2(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppEndpoint, error)
	GetPodWorkloads(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppWorkload, error)
	DeletePod(ctx context.Context, clusterID string, namespace string, podName string) error
}

type manager struct {
	configPath string
	admClient  adm.Client
}

type serviceProxyURL struct {
	domainName       string
	projectID        string
	clusterID        string
	serviceNamespace string
	serviceName      string
	servicePort      string
	serviceProtocol  string
}

func (m *manager) DeletePod(ctx context.Context, clusterID string, namespace string, podName string) error {
	k8sClient, err := getK8sClient(ctx, clusterID, m.admClient, m.configPath)
	if err != nil {
		log.Warnw("Failed to create a k8s client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return err
	}

	err = k8sClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		log.Warnw("Failed to delete pod", dazl.String("PodName", podName), dazl.Error(err))
		return k8serrors.K8sToTypedError(err)
	}

	return nil
}

func checkOwnerAnnotations(ctx context.Context, k8sClient kubernetes.Interface, dynK8sClient dynamic.Interface, meta *metav1.ObjectMeta, namespace, appID string) (bool, error) {
	if value, exist := meta.GetAnnotations()[AnnotationKeyForAppID]; exist && value == appID {
		return true, nil
	}
	for _, ownerRef := range meta.GetOwnerReferences() {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			switch ownerRef.Kind {
			case "ReplicaSet":
				return checkOwnerReplicaSet(ctx, k8sClient, dynK8sClient, ownerRef, namespace, appID)
			case "DaemonSet":
				return checkOwnerDaemonSet(ctx, k8sClient, dynK8sClient, ownerRef, namespace, appID)
			case "StatefulSet":
				return checkOwnerStatefulSet(ctx, k8sClient, dynK8sClient, ownerRef, namespace, appID)
			case "SriovOperatorConfig":
				return checkOwnerSriovOperatorConfig(ctx, k8sClient, dynK8sClient, ownerRef, namespace, appID)
			}
			return false, nil
		}
	}
	return false, nil
}

func checkOwnerReplicaSet(ctx context.Context, k8sClient kubernetes.Interface, dynK8sClient dynamic.Interface, ownerRef metav1.OwnerReference, namespace, appID string) (bool, error) {
	rs, err := k8sClient.AppsV1().ReplicaSets(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Warnw("Failed to get list of ReplicaSets", dazl.Error(err))
		return false, k8serrors.K8sToTypedError(err)
	}
	return checkOwnerAnnotations(ctx, k8sClient, dynK8sClient, &rs.ObjectMeta, namespace, appID)
}

func checkOwnerDaemonSet(ctx context.Context, k8sClient kubernetes.Interface, dynK8sClient dynamic.Interface, ownerRef metav1.OwnerReference, namespace, appID string) (bool, error) {
	ds, err := k8sClient.AppsV1().DaemonSets(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Warnw("Failed to get list of DaemonSets", dazl.Error(err))
		return false, k8serrors.K8sToTypedError(err)
	}
	return checkOwnerAnnotations(ctx, k8sClient, dynK8sClient, &ds.ObjectMeta, namespace, appID)
}

func checkOwnerStatefulSet(ctx context.Context, k8sClient kubernetes.Interface, dynK8sClient dynamic.Interface, ownerRef metav1.OwnerReference, namespace, appID string) (bool, error) {
	ss, err := k8sClient.AppsV1().StatefulSets(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Warnw("Failed to get list of StatefulSets", dazl.Error(err))
		return false, k8serrors.K8sToTypedError(err)
	}
	return checkOwnerAnnotations(ctx, k8sClient, dynK8sClient, &ss.ObjectMeta, namespace, appID)
}

// We have to check the sriovoperatorconfig because the SRIOV cluster extension
// creates pods using the operator pattern. These pods are not accounted for by
// fleet and hence would not be reported in the UI.
func checkOwnerSriovOperatorConfig(ctx context.Context, k8sClient kubernetes.Interface, dynK8sClient dynamic.Interface, ownerRef metav1.OwnerReference, namespace, appID string) (bool, error) {
	gvr := schema.GroupVersionResource{Group: "sriovnetwork.openshift.io", Version: "v1", Resource: "sriovoperatorconfigs"}
	us, err := dynK8sClient.Resource(gvr).Namespace(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Warnw("Failed to get list of sriovoperatorconfigs", dazl.Error(err))
		return false, k8serrors.K8sToTypedError(err)
	}
	// unstructured.Unstructured does not have a ObjectMeta field
	meta := metav1.ObjectMeta{
		Annotations:     us.GetAnnotations(),
		OwnerReferences: us.GetOwnerReferences(),
	}
	return checkOwnerAnnotations(ctx, k8sClient, dynK8sClient, &meta, namespace, appID)
}

func (m *manager) GetPodWorkloads(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppWorkload, error) {
	log.Infow("Getting Pod Workloads", dazl.String("appID", appID), dazl.String("clusterID", clusterID))
	podWorkloads := make([]*resourceapiv2.AppWorkload, 0)

	k8sClient, err := getK8sClient(ctx, clusterID, m.admClient, m.configPath)
	if err != nil {
		log.Warnw("Failed to create a k8s client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return nil, err
	}

	dynK8sClient, err := getDynamicK8sClient(ctx, clusterID, m.admClient, m.configPath)
	if err != nil {
		log.Warnw("Failed to create a dynamic k8s client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return nil, err
	}

	appNamespace, err := m.admClient.GetAppNamespace(ctx, appID)
	if err != nil {
		log.Warnw("Failed to get application namespace", dazl.String("AppID", appID), dazl.Error(err))
		return nil, err
	}

	excludeLabelSelector := fmt.Sprintf("%s!=%s", LabelKeyForVirtHandler, "virt-launcher")
	pods, err := k8sClient.CoreV1().Pods(appNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: excludeLabelSelector,
	})
	if err != nil {
		log.Warnw("Failed to get list of pods", dazl.String("AppID", appID), dazl.Error(err))
		return nil, k8serrors.K8sToTypedError(err)
	}

	var appPodList []corev1.Pod
	for _, pod := range pods.Items {
		isOwned, err := checkOwnerAnnotations(ctx, k8sClient, dynK8sClient, &pod.ObjectMeta, pod.Namespace, appID)
		if err != nil {
			log.Warnw("Failed to check owner reference", dazl.String("PodName", pod.Name), dazl.Error(err))
			return nil, err
		}
		if isOwned {
			appPodList = append(appPodList, pod)
		}
	}

	for _, pod := range appPodList {
		podWorkLoad := &resourceapiv2.AppWorkload{
			Type:       resourceapiv2.AppWorkload_TYPE_POD,
			Id:         string(pod.ObjectMeta.UID),
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			CreateTime: timestamppb.New(pod.ObjectMeta.CreationTimestamp.Time),
		}

		containers := pod.Spec.Containers
		podResource := &resourceapiv2.AppWorkload_Pod{
			Pod: &resourceapiv2.Pod{
				Status: convertPodPhase(pod.Status.Phase),
			},
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				podWorkLoad.WorkloadReady = true
			} else if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
				podWorkLoad.WorkloadReady = false
			}
		}

		for index, container := range containers {
			containerResource := &resourceapiv2.Container{
				Name:      container.Name,
				ImageName: container.Image,
			}
			if index < len(pod.Status.ContainerStatuses) {
				containerStatus := pod.Status.ContainerStatuses[index]
				containerResource.RestartCount = containerStatus.RestartCount
				containerResource.Status = convertContainerStatus(containerStatus.State)
			} else {
				log.Warnw("Container status not found for container", dazl.String("ContainerName", container.Name))
			}
			podResource.Pod.Containers = append(podResource.Pod.Containers, containerResource)
		}
		podWorkLoad.Workload = podResource
		podWorkloads = append(podWorkloads, podWorkLoad)
	}

	return podWorkloads, nil

}

func (m *manager) GetAppEndpointsV2(ctx context.Context, appID string, clusterID string) ([]*resourceapiv2.AppEndpoint, error) {
	k8sClient, err := getK8sClient(ctx, clusterID, m.admClient, m.configPath)
	if err != nil {
		log.Warnw("Failed to create a k8s client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return nil, err
	}
	appEndpoints := make([]*resourceapiv2.AppEndpoint, 0)
	appNamespace, err := m.admClient.GetAppNamespace(ctx, appID)
	if err != nil {
		log.Warnw("Failed to get application namespace", dazl.String("AppID", appID), dazl.Error(err))
		return nil, err
	}

	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		// Return error since ActiveProjectID is mandatory
		log.Errorw("Failed to get active project ID", dazl.Error(err))
		return nil, err
	}

	serviceEndpoints, err := getServiceAppEndpointsV2(ctx, k8sClient, activeProjectID, appID, appNamespace, clusterID)
	if err != nil {
		log.Warnw("Failed to get application service endpoints", dazl.String("AppID", appID), dazl.Error(err))
		return nil, err
	}
	appEndpoints = append(appEndpoints, serviceEndpoints...)

	ingressEndpoints, err := getIngressAppEndpointsV2(ctx, k8sClient, appID, appNamespace)
	if err != nil {
		log.Warnw("Failed to get application ingress endpoints", dazl.String("AppID", appID), dazl.Error(err))
		return nil, err
	}

	appEndpoints = append(appEndpoints, ingressEndpoints...)

	return appEndpoints, nil

}

func getServicePorts(service corev1.Service, projectID, clusterID string) ([]*resourceapiv2.Port, error) {
	ports := make([]*resourceapiv2.Port, 0)
	domainName := os.Getenv(serviceProxyDomainName)
	for _, svcPort := range service.Spec.Ports {
		port := &resourceapiv2.Port{
			Name:     svcPort.Name,
			Value:    svcPort.Port,
			Protocol: string(svcPort.Protocol),
		}

		if serviceProxyPorts, ok := service.Annotations[ServiceProxyPortAnnotation]; ok {
			portProtocolPairs := strings.Split(serviceProxyPorts, ",")
			for _, portProtoclPair := range portProtocolPairs {
				trimmedProtocolPair := strings.TrimSpace(portProtoclPair)
				parts := strings.Split(trimmedProtocolPair, ":")
				if len(parts) == 2 {
					protocol, servicePort := parts[0], parts[1]
					servicePortInt, err := strconv.ParseInt(servicePort, 10, 32)
					if err != nil {
						log.Warnw("Failed to parse service port", dazl.Error(err))
						return nil, err
					}
					if int32(servicePortInt) == svcPort.Port {
						svcProxyURL := serviceProxyURL{
							projectID:        projectID,
							clusterID:        clusterID,
							serviceName:      service.Name,
							serviceNamespace: service.Namespace,
							servicePort:      servicePort,
							serviceProtocol:  protocol,
							domainName:       domainName,
						}

						url := createServiceProxyURL(svcProxyURL)
						port.ServiceProxyUrl = url
					}

				} else if len(parts) == 1 {
					servicePort := parts[0]
					servicePortInt, err := strconv.ParseInt(servicePort, 10, 32)
					if err != nil {
						log.Warnw("Failed to parse service port", dazl.Error(err))
						return nil, err
					}
					if int32(servicePortInt) == svcPort.Port {
						svcProxyURL := serviceProxyURL{
							projectID:        projectID,
							clusterID:        clusterID,
							serviceName:      service.Name,
							serviceNamespace: service.Namespace,
							servicePort:      servicePort,
							domainName:       domainName,
						}

						url := createServiceProxyURL(svcProxyURL)
						port.ServiceProxyUrl = url
					}
				}

			}
		}
		ports = append(ports, port)
	}

	return ports, nil
}

func getServiceAppEndpointsV2(ctx context.Context, k8sClient kubernetes.Interface, projectID string, appID string, appNamespace string, clusterID string) ([]*resourceapiv2.AppEndpoint, error) {
	endpoints := make([]*resourceapiv2.AppEndpoint, 0)
	services, err := k8sClient.CoreV1().Services(appNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, k8serrors.K8sToTypedError(err)
	}

	for _, service := range services.Items {
		serviceAppID := service.Annotations[AnnotationKeyForAppID]
		if serviceAppID == appID {
			ports, err := getServicePorts(service, projectID, clusterID)
			if err != nil {
				return nil, err
			}

			// Get list of k8s endpoints for a service
			k8sEndpoints, err := k8sClient.CoreV1().Endpoints(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
			if err != nil {
				return nil, k8serrors.K8sToTypedError(err)
			}
			appEndpointStatus := &resourceapiv2.EndpointStatus{}

			// If there is no endpoints associated with the service then the service is not ready
			if len(k8sEndpoints.Subsets) == 0 {
				appEndpointStatus.State = resourceapiv2.EndpointStatus_STATE_NOT_READY
			} else if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				if len(service.Status.LoadBalancer.Ingress) == 0 {
					appEndpointStatus.State = resourceapiv2.EndpointStatus_STATE_NOT_READY
				} else {
					appEndpointStatus.State = resourceapiv2.EndpointStatus_STATE_READY
				}
			} else {
				appEndpointStatus.State = resourceapiv2.EndpointStatus_STATE_READY
			}
			appEndpoint := &resourceapiv2.AppEndpoint{
				Id:             string(service.ObjectMeta.UID),
				Name:           service.Name,
				Ports:          ports,
				EndpointStatus: appEndpointStatus,
			}

			if hostNameValue, ok := service.Annotations[ExternalDNSAnnotation]; ok {
				hostnameSlice := strings.Split(hostNameValue, ",")
				var fqdns []*resourceapiv2.Fqdn
				for _, hostNameSliceValue := range hostnameSlice {
					fqdns = append(fqdns, &resourceapiv2.Fqdn{
						Fqdn: hostNameSliceValue,
					})
				}

				appEndpoint.Fqdns = fqdns
			}

			endpoints = append(endpoints, appEndpoint)
		}
	}
	return endpoints, nil

}

func getIngressAppEndpointsV2(ctx context.Context, k8sClient kubernetes.Interface, appID string, appNamespace string) ([]*resourceapiv2.AppEndpoint, error) {
	appEndpoints := make([]*resourceapiv2.AppEndpoint, 0)
	ingresses, err := k8sClient.NetworkingV1().Ingresses(appNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, k8serrors.K8sToTypedError(err)
	}

	for _, ingress := range ingresses.Items {
		ingressAppID := ingress.Annotations[AnnotationKeyForAppID]
		// If ingress is associated with the given appID then processed to extract the required info
		if ingressAppID == appID {
			appEndpoint := &resourceapiv2.AppEndpoint{}
			endpointStatus := &resourceapiv2.EndpointStatus{}
			appEndpoint.Id = string(ingress.ObjectMeta.UID)
			appEndpoint.Name = ingress.Name

			/*
				For ingresses, you can optionally force ExternalDNS to create records
				based on either the hosts specified or the external-dns.alpha.kubernetes.io/hostname annotation.
				This behavior is controlled by setting the external-dns.alpha.kubernetes.io/ingress-hostname-source annotation
				on that ingress to either defined-hosts-only or annotation-only.
			*/
			hostnameSource, ok := ingress.Annotations[IngressHostnameSourceAnnotation]
			if ok && hostnameSource == AnnotationOnly {
				hostnameSlice := strings.Split(ingress.Annotations[ExternalDNSAnnotation], ",")
				var fqdns []*resourceapiv2.Fqdn
				for _, hostNameSliceValue := range hostnameSlice {
					fqdns = append(fqdns, &resourceapiv2.Fqdn{
						Fqdn: hostNameSliceValue,
					})
				}
				appEndpoint.Fqdns = fqdns
			} else if ok && hostnameSource == DefinedHostOnly {
				if len(ingress.Spec.Rules) > 0 {
					// TODO do we need to go through all rules and assign hosts?
					appEndpoint.Fqdns = []*resourceapiv2.Fqdn{
						{
							Fqdn: ingress.Spec.Rules[0].Host,
						},
					}

				}
			}

			for _, rule := range ingress.Spec.Rules {
				if rule.HTTP != nil {
					for _, path := range rule.HTTP.Paths {
						port := &resourceapiv2.Port{}
						port.Name = path.Backend.Service.Port.Name

						if ingress.Spec.TLS != nil {
							port.Protocol = "HTTPS"
							port.Value = 443
						} else {
							port.Protocol = "HTTP"
							port.Value = 80
						}
						appEndpoint.Ports = append(appEndpoint.Ports, port)
					}
				}

			}

			if len(ingress.Status.LoadBalancer.Ingress) == 0 {
				endpointStatus.State = resourceapiv2.EndpointStatus_STATE_NOT_READY
			} else {
				endpointStatus.State = resourceapiv2.EndpointStatus_STATE_READY
			}
			appEndpoint.EndpointStatus = endpointStatus
			appEndpoints = append(appEndpoints, appEndpoint)
		}
	}
	return appEndpoints, nil
}
