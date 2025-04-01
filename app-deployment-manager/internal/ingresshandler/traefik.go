// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package ingresshandler

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	traefikv1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikcontainous/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
)

// TraefikIngressHandler implements IngressHandler
type traefikIngressHandler struct {
	scheme *runtime.Scheme
	client client.Client
}

func (t *traefikIngressHandler) Init() {
	// Must panic on non-nil errors
	utilruntime.Must(traefikv1alpha1.AddToScheme(t.scheme))
}

func (t *traefikIngressHandler) GetRoute(ctx context.Context, key client.ObjectKey, _ ...client.GetOption) (client.Object, error) {
	ingressRoute := &traefikv1alpha1.IngressRoute{}
	err := t.client.Get(ctx, key, ingressRoute)
	return ingressRoute, err
}

func (t *traefikIngressHandler) CreateRoute(ctx context.Context, apiExtension *v1beta1.APIExtension, endpoint v1beta1.ProxyEndpoint) error {
	// Create middleware
	middleware := &traefikv1alpha1.Middleware{ObjectMeta: metav1.ObjectMeta{
		Name:      endpoint.ServiceName,
		Namespace: apiExtension.Namespace,
		Annotations: map[string]string{
			v1beta1.APIExtensionAnnotation: apiExtension.Name,
		},
	},
		Spec: traefikv1alpha1.MiddlewareSpec{
			ReplacePath: &dynamic.ReplacePath{
				Path: fmt.Sprintf("/client/%s/%s/%s", apiExtension.Name, endpoint.Scheme, endpoint.Backend),
			},
		},
	}

	if err := ctrl.SetControllerReference(apiExtension, middleware, t.scheme); err != nil {
		// Ingress object is owned by the associated apiextension
		// and cannot be shared by other resources
		// This middleware is removed by GC when owner apiextension is deleted
		return fmt.Errorf("failed to set owner for traefik ingress middleware %q (%v)",
			endpoint.ServiceName, err)
	}

	if err := t.client.Create(ctx, middleware); err != nil {
		return fmt.Errorf("failed to create traefik ingress middleware %q (%v)",
			endpoint.ServiceName, err)
	}

	cfg := config.GetAPIExtensionConfig()
	apiProxyNamespace := apiExtension.Namespace
	if cfg.APIProxyNamespace != "" {
		apiProxyNamespace = cfg.APIProxyNamespace
	}

	// Create ingress route
	ingressRoute := &traefikv1alpha1.IngressRoute{ObjectMeta: metav1.ObjectMeta{
		Name:      endpoint.ServiceName,
		Namespace: apiExtension.Namespace,
		Annotations: map[string]string{
			v1beta1.APIExtensionAnnotation: apiExtension.Name,
		},
	},
		Spec: traefikv1alpha1.IngressRouteSpec{
			EntryPoints: []string{
				"websecure",
				"web",
			},
			Routes: []traefikv1alpha1.Route{
				{
					Match: "HostRegexp(`{any:.+}`) && " + getPathPrefix(apiExtension.Spec.APIGroup, endpoint.Path),
					Services: []traefikv1alpha1.Service{
						{
							LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{
								Kind:      "Service",
								Name:      cfg.APIProxyService,
								Namespace: apiProxyNamespace,
								Port:      intstr.FromInt32(cfg.APIProxyPort),
								Scheme:    "http",
							},
						},
					},
					Middlewares: []traefikv1alpha1.MiddlewareRef{
						{
							Name:      endpoint.ServiceName,
							Namespace: apiExtension.Namespace,
						},
					},
					Kind: "Rule",
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(apiExtension, ingressRoute, t.scheme); err != nil {
		// Ingress object is owned by the associated apiextension
		// and cannot be shared by other resources
		// This ingress route is removed by GC when owner apiextension is deleted
		return fmt.Errorf("failed to set owner for traefik ingress route %q (%v)",
			endpoint.ServiceName, err)
	}

	if err := t.client.Create(ctx, ingressRoute); err != nil {
		return fmt.Errorf("failed to create traefik ingress route %q (%v)",
			endpoint.ServiceName, err)
	}

	return nil
}

func getPathPrefix(apiGroup v1beta1.APIGroup, path string) string {
	return fmt.Sprintf("PathPrefix(`/%s.%s/%s/%s`)",
		apiGroup.Name,
		config.GetAPIExtensionConfig().APIGroupDomain,
		apiGroup.Version,
		path)
}
