// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"errors"
	"fmt"
	admv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
	"time"
)

const defaultResyncPeriod = time.Hour * 24

func newDeploymentServiceSource(client clusterclient.Client, cache cache.Cache, handler handler.TypedEventHandler[*admv1beta1.DeploymentCluster, reconcile.Request]) *deploymentServiceSource {
	return &deploymentServiceSource{
		Client:  client,
		Cache:   cache,
		Handler: handler,
	}
}

type deploymentServiceSource struct {
	Client      clusterclient.Client
	Cache       cache.Cache
	Handler     handler.TypedEventHandler[*admv1beta1.DeploymentCluster, reconcile.Request]
	startedErr  chan error
	startCancel func()
}

func (s *deploymentServiceSource) Start(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	// cache.GetInformer will block until its context is cancelled if the cache was already started and it can not
	// sync that informer (most commonly due to RBAC issues).
	ctx, s.startCancel = context.WithCancel(ctx)
	s.startedErr = make(chan error)
	go func() {
		var (
			i       cache.Informer
			lastErr error
		)

		// Tries to get an informer until it returns true,
		// an error or the specified context is cancelled or expired.
		if err := wait.PollUntilContextCancel(ctx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
			// Lookup the Informer from the Cache and add an EventHandler which populates the Queue
			i, lastErr = s.Cache.GetInformer(ctx, &admv1beta1.DeploymentCluster{})
			if lastErr != nil {
				kindMatchErr := &meta.NoKindMatchError{}
				switch {
				case errors.As(lastErr, &kindMatchErr):
					log.Error(lastErr, "if kind is a CRD, it should be installed before calling Start",
						"kind", kindMatchErr.GroupKind)
				case runtime.IsNotRegisteredError(lastErr):
					log.Error(lastErr, "kind must be registered to the Scheme")
				default:
					log.Error(lastErr, "failed to get informer from cache")
				}
				return false, nil // Retry.
			}
			return true, nil
		}); err != nil {
			if lastErr != nil {
				s.startedErr <- fmt.Errorf("failed to get informer from cache: %w", lastErr)
				return
			}
			s.startedErr <- err
			return
		}

		_, err := i.AddEventHandler(newDeploymentClusterEventHandler(ctx, s, queue, s.Handler))
		if err != nil {
			s.startedErr <- err
			return
		}
		if !s.Cache.WaitForCacheSync(ctx) {
			log.Error("Failed to sync cache for %T", &admv1beta1.DeploymentCluster{})
			s.startedErr <- errors.New("cache did not sync")
		}
		close(s.startedErr)
	}()
	return nil
}

// WaitForSync implements SyncingSource to allow controllers to wait with starting
// workers until the cache is synced.
func (s *deploymentServiceSource) WaitForSync(ctx context.Context) error {
	select {
	case err := <-s.startedErr:
		return err
	case <-ctx.Done():
		s.startCancel()
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return fmt.Errorf("timed out waiting for cache to be synced for Kind %T", &admv1beta1.DeploymentCluster{})
	}
}

var _ source.SyncingSource = (*deploymentServiceSource)(nil)

func newDeploymentClusterEventHandler(ctx context.Context, source *deploymentServiceSource, queue workqueue.TypedRateLimitingInterface[reconcile.Request], handler handler.TypedEventHandler[*admv1beta1.DeploymentCluster, reconcile.Request]) clientcache.ResourceEventHandler {
	return &deploymentClusterHandler{
		deploymentServiceSource: source,
		ctx:                     ctx,
		queue:                   queue,
		handler:                 handler,
		serviceSources:          make(map[corev1.ObjectReference]*serviceSource),
		serviceCancelFuncs:      make(map[corev1.ObjectReference]context.CancelFunc),
	}
}

type deploymentClusterHandler struct {
	*deploymentServiceSource
	ctx                context.Context
	queue              workqueue.TypedRateLimitingInterface[reconcile.Request]
	handler            handler.TypedEventHandler[*admv1beta1.DeploymentCluster, reconcile.Request]
	serviceSources     map[corev1.ObjectReference]*serviceSource
	serviceCancelFuncs map[corev1.ObjectReference]context.CancelFunc
	mu                 sync.Mutex
}

func (h *deploymentClusterHandler) OnAdd(obj interface{}, _ bool) {
	if deploymentCluster, ok := obj.(*admv1beta1.DeploymentCluster); ok {
		_ = h.startWatching(deploymentCluster)
	} else {
		log.Error("invalid object")
	}
}

func (h *deploymentClusterHandler) OnUpdate(oldObjRaw, newObjRaw interface{}) {
	oldDeploymentCluster, ok := oldObjRaw.(*admv1beta1.DeploymentCluster)
	if !ok {
		return
	}

	newDeploymentCluster, ok := newObjRaw.(*admv1beta1.DeploymentCluster)
	if !ok {
		return
	}

	if oldDeploymentCluster.Spec.ClusterID != newDeploymentCluster.Spec.ClusterID {
		_ = h.stopWatching(oldDeploymentCluster)
		_ = h.startWatching(newDeploymentCluster)
	}
}

func (h *deploymentClusterHandler) OnDelete(obj interface{}) {
	if deploymentCluster, ok := obj.(*admv1beta1.DeploymentCluster); ok {
		_ = h.stopWatching(deploymentCluster)
	} else {
		log.Error("invalid object")
	}
}

func (h *deploymentClusterHandler) startWatching(deploymentCluster *admv1beta1.DeploymentCluster) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	clusterRef := corev1.ObjectReference{
		Namespace: deploymentCluster.Spec.Namespace,
		Name:      deploymentCluster.Spec.ClusterID,
	}
	source, ok := h.serviceSources[clusterRef]
	start := !ok
	if start {
		ctx, cancel := context.WithCancel(h.ctx)
		defer cancel()

		var projectID clusterclient.ProjectID
		clusterID := clusterclient.ClusterID(deploymentCluster.Spec.ClusterID)
		projectID = clusterclient.ProjectID(deploymentCluster.Labels[utils.AppOrchProjectIDLabel])

		config, err := h.Client.GetClusterConfig(ctx, clusterID, projectID)
		if err != nil {
			log.Error(err)
			return err
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Error(err)
			return err
		}

		factory := informers.NewSharedInformerFactory(clientset, defaultResyncPeriod)

		source = newServiceSource(clusterRef, factory)
		h.serviceSources[clusterRef] = source
	}

	source.addHandler(deploymentCluster, func(service *corev1.Service) {
		log.Infof("Handling %s event for %s", service.GetObjectKind(), service.GetName())
		annotations := service.GetAnnotations()
		if annotations == nil || annotations[networkExposeServiceAnnotation] == "" {
			log.Debugf("Ignoring %s event for %s as %s annotation is missing", service.GetObjectKind(), service.GetName(), networkExposeServiceAnnotation)
			return
		}
		generic := event.TypedGenericEvent[*admv1beta1.DeploymentCluster]{
			Object: deploymentCluster,
		}
		log.Debugf("Publishing generic %s event for %s", deploymentCluster.GetObjectKind(), deploymentCluster.GetName())
		ctx, cancel := context.WithCancel(h.ctx)
		defer cancel()
		h.handler.Generic(ctx, generic, h.queue)
	})

	if start {
		ctx, cancel := context.WithCancel(h.ctx)
		h.serviceCancelFuncs[clusterRef] = cancel
		return source.Start(ctx, h.queue)
	}
	return nil
}

func (h *deploymentClusterHandler) stopWatching(deploymentCluster *admv1beta1.DeploymentCluster) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	clusterRef := corev1.ObjectReference{
		Namespace: deploymentCluster.Spec.Namespace,
		Name:      deploymentCluster.Spec.ClusterID,
	}
	source, ok := h.serviceSources[clusterRef]
	if !ok {
		return nil
	}

	if source.removeHandler(deploymentCluster) == 0 {
		delete(h.serviceSources, clusterRef)
		serviceCancelFunc, ok := h.serviceCancelFuncs[clusterRef]
		if ok {
			serviceCancelFunc()
			delete(h.serviceCancelFuncs, clusterRef)
		}
	}
	return nil
}

var _ clientcache.ResourceEventHandler = (*deploymentClusterHandler)(nil)

func newServiceSource(clusterRef corev1.ObjectReference, factory informers.SharedInformerFactory) *serviceSource {
	return &serviceSource{
		clusterRef: clusterRef,
		factory:    factory,
		handlers:   make(map[types.UID]func(resource *corev1.Service)),
	}
}

type serviceSource struct {
	clusterRef corev1.ObjectReference
	factory    informers.SharedInformerFactory
	handlers   map[types.UID]func(resource *corev1.Service)
	mu         sync.RWMutex
}

func (s *serviceSource) addHandler(resource *admv1beta1.DeploymentCluster, handler func(resource *corev1.Service)) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[resource.GetUID()] = handler
	return len(s.handlers)
}

func (s *serviceSource) removeHandler(resource *admv1beta1.DeploymentCluster) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.handlers, resource.GetUID())
	return len(s.handlers)
}

func (s *serviceSource) getHandlers() []func(resource *corev1.Service) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	handlers := make([]func(resource *corev1.Service), 0, len(s.handlers))
	for _, handler := range s.handlers {
		handlers = append(handlers, handler)
	}
	return handlers
}

func (s *serviceSource) handle(object client.Object) {
	if service, ok := object.(*corev1.Service); ok {
		for _, handler := range s.getHandlers() {
			handler(service)
		}
	}
}

func (s *serviceSource) Start(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	log.Infof("Starting Service Source %s", s.clusterRef)
	source := &source.Informer{
		Informer: s.factory.Core().V1().Services().Informer(),
		Handler: handler.Funcs{
			CreateFunc: func(_ context.Context, event event.TypedCreateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				s.handle(event.Object)
			},
			UpdateFunc: func(_ context.Context, event event.TypedUpdateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				s.handle(event.ObjectNew)
			},
			DeleteFunc: func(_ context.Context, event event.TypedDeleteEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				s.handle(event.Object)
			},
			GenericFunc: func(_ context.Context, event event.TypedGenericEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				s.handle(event.Object)
			},
		},
	}
	return source.Start(ctx, queue)
}

var _ source.Source = (*serviceSource)(nil)
