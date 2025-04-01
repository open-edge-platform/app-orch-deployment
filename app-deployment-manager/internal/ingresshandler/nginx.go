// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ingresshandler

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nginxRewriteAnnotation = "nginx.ingress.kubernetes.io/rewrite-target"
)

var nginxClassName = "nginx"

// NginxIngressHandler implements IngressHandler
type nginxIngressHandler struct {
	scheme *runtime.Scheme
	client client.Client
}

func (n *nginxIngressHandler) Init() {
	// Do nothing
}

func (n *nginxIngressHandler) GetRoute(ctx context.Context, key client.ObjectKey, _ ...client.GetOption) (client.Object, error) {
	ingress := &networkv1.Ingress{}
	err := n.client.Get(ctx, key, ingress)
	return ingress, err
}

func (n *nginxIngressHandler) CreateRoute(ctx context.Context, apiExtension *v1beta1.APIExtension, endpoint v1beta1.ProxyEndpoint) error {
	ingress := &networkv1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Name:      endpoint.ServiceName,
		Namespace: apiExtension.Namespace,
		Annotations: map[string]string{
			v1beta1.APIExtensionAnnotation: apiExtension.Name,
			nginxRewriteAnnotation:         fmt.Sprintf("/client/%s/%s/$2", apiExtension.Name, endpoint.Backend),
		},
	},
		Spec: networkv1.IngressSpec{
			IngressClassName: &nginxClassName,
			Rules: []networkv1.IngressRule{
				n.getNginxIngressRule(apiExtension.Spec.APIGroup, endpoint.Path),
			},
		},
	}

	if err := ctrl.SetControllerReference(apiExtension, ingress, n.scheme); err != nil {
		// Ingress object is owned by the associated apiextension
		// and cannot be shared by other resources
		// This ingress is removed by GC when owner apiextension is deleted
		return fmt.Errorf("failed to set owner for ingress %q (%v)", endpoint.ServiceName, err)
	}

	if err := n.client.Create(ctx, ingress); err != nil {
		return fmt.Errorf("failed to create ingress %q (%v)", endpoint.ServiceName, err)
	}

	return nil
}

func (n *nginxIngressHandler) getNginxIngressRule(apiGroup v1beta1.APIGroup, path string) networkv1.IngressRule {
	cfg := config.GetAPIExtensionConfig()
	pathType := networkv1.PathTypePrefix

	return networkv1.IngressRule{
		IngressRuleValue: networkv1.IngressRuleValue{
			HTTP: &networkv1.HTTPIngressRuleValue{
				Paths: []networkv1.HTTPIngressPath{
					{
						PathType: &pathType,
						Path:     n.getPath(apiGroup, path),
						Backend: networkv1.IngressBackend{
							Service: &networkv1.IngressServiceBackend{
								Name: cfg.APIProxyService,
								Port: networkv1.ServiceBackendPort{
									Number: cfg.APIProxyPort,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (n *nginxIngressHandler) getPath(apiGroup v1beta1.APIGroup, path string) string {
	return fmt.Sprintf("/%s.%s/%s/%s(/|$)(.*)",
		apiGroup.Name,
		config.GetAPIExtensionConfig().APIGroupDomain,
		apiGroup.Version,
		path)
}
