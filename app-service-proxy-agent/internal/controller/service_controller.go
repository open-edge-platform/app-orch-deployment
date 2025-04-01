// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ServiceProxyAnnotation = "service-proxy.app.orchestrator.io/ports"
)

var (
	setupLog                        = ctrl.Log.WithName("Controller")
	serviceProxyAnnotationPredicate = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// pass only when the object has service proxy annotation
			return hasAnnotationKey(e.Object, ServiceProxyAnnotation)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// pass only when the object has service proxy annotation
			return hasAnnotationKey(e.Object, ServiceProxyAnnotation)
		},
		UpdateFunc: func(_ event.UpdateEvent) bool {
			// FIXME: pass when the object has service proxy annotation and
			// the annotation has changed
			return false
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
	}
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	saName      string
	saNamespace string
}

//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services/proxy,verbs=get;list;watch;create;
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles/finalizers,verbs=update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager, saName, saNamspace string) error {
	r.saName = saName
	r.saNamespace = saNamspace

	_, err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}, builder.WithPredicates(serviceProxyAnnotationPredicate)).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Build(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up service reconciler")
	}

	return nil
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	s := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, s); err != nil {
		// Error reading the object, requeue the request, unless error is "not found"
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileRole(ctx, s); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileRoleBinding(ctx, s); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileRole(ctx context.Context, s *corev1.Service) error {
	allowedResources, err := getResourceNames(s)
	if err != nil || len(allowedResources) == 0 {
		// No valid annotation value found, return early and do not requeue
		return err
	}

	rule := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{""},
			Verbs:         []string{"get", "list", "watch", "create"},
			Resources:     []string{"services/proxy"},
			ResourceNames: allowedResources,
		},
	}

	role := &rbacv1.Role{}
	err = r.Get(ctx, client.ObjectKey{Name: getRoleName(s), Namespace: s.Namespace}, role)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get Role %s(%v)", getRoleName(s), err)
	}

	if err == nil {
		// Role already exists, update existing one
		role.Rules = rule
		if err := r.Client.Update(ctx, role); err != nil {
			return fmt.Errorf("Role update failed(%v)", err)
		}
	} else {
		// Create a Role that allows service/proxy access for the requested service ports
		role = &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      getRoleName(s),
		},
			Rules: rule,
		}

		// Set the Service as owner of the Role for the garbage collection
		if err := ctrl.SetControllerReference(s, role, r.Scheme); err != nil {
			return err
		}

		if err := r.Client.Create(ctx, role); err != nil {
			return fmt.Errorf("Role creation failed(%v)", err)
		}
	}

	return nil
}

func (r *ServiceReconciler) reconcileRoleBinding(ctx context.Context, s *corev1.Service) error {
	roleBinding := &rbacv1.RoleBinding{}
	err := r.Get(ctx,
		client.ObjectKey{
			Name:      getRoleName(s),
			Namespace: s.Namespace},
		roleBinding)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get RoleBinding %s(%v)", getRoleName(s), err)
	}

	if err == nil {
		// RoleBinding already exists, return early
		return nil
	}

	// Create a RoleBinding with the role and service account
	rolebinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      getRoleName(s),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     getRoleName(s),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.GroupKind,
				Name: "system:authenticated",
			},
		},
	}
	// Set the Service as owner of the RoleBinding for the garbage collection
	if err := ctrl.SetControllerReference(s, rolebinding, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Create(ctx, rolebinding); err != nil {
		return fmt.Errorf("RoleBinding creation failed (%v)", err)
	}

	return nil
}

func getRoleName(s *corev1.Service) string {
	return fmt.Sprintf("orchestrator:app:%s:%s", s.Namespace, s.Name)
}

func getResourceNames(s *corev1.Service) ([]string, error) {
	resources := make([]string, 0)

	value, ok := s.Annotations[ServiceProxyAnnotation]
	if !ok {
		return resources, fmt.Errorf("annotation %s is not found", ServiceProxyAnnotation)
	}

	for _, p := range strings.Split(value, ",") {
		protoPort := strings.Split(p, ":")
		port := protoPort[0]
		protocol := ""

		if len(protoPort) > 2 {
			setupLog.Info(fmt.Sprintf("invalid service annotation port format [ %v ] - accepted format protocol:port", p))
			continue
		}

		// Include protocol and port in the resource name. This is required when creating
		// k8s RBAC roles.
		if len(protoPort) == 2 {
			protocol = protoPort[0]
			port = protoPort[1]
		}

		intPort, err := strconv.Atoi(port)
		if err != nil || 1 > intPort || intPort > 65535 {
			continue
		}
		// If resource name doesn't match name ie protocol:s.Name:port
		// then service link will return 403 Forbidden
		if protocol != "" {
			resources = append(resources, fmt.Sprintf("%s:%s:%s", protocol, s.Name, port))
		} else {
			resources = append(resources, fmt.Sprintf("%s:%s", s.Name, port))
		}
	}

	return resources, nil
}

func hasAnnotationKey(obj client.Object, annotationKey string) bool {
	// Return early if no annotationKey was set.
	if annotationKey == "" {
		return false
	}

	_, ok := obj.GetAnnotations()[annotationKey]
	return ok
}
