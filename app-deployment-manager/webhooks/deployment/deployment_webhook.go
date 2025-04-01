// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
)

const (
	appRefFilter = ".metadata.deployment-package-ref"
)

// SetupWebhookWithManager sets up Deployment webhooks.
func (webhook *Deployment) SetupWebhookWithManager(mgr ctrl.Manager) (err error) {
	webhook.catalogclient, err = catalogclient.NewCatalogClient()
	if err != nil {
		return err
	}

	webhook.vaultAuthClient, err = auth.NewVaultAuth(utils.GetKeycloakServiceEndpoint(), utils.GetSecretServiceEndpoint(), utils.GetServiceAccount())
	if err != nil {
		return err
	}

	deleteRepo, ok := os.LookupEnv("GITEA_DELETE_REPO_ON_TERMINATE")
	if !ok || deleteRepo == "true" {
		webhook.deleteGitRepo = true
	}

	// Add FieldIndexter that adds an index with .metadata.deploymentPackage
	// field name and <deployment package name>-<version> as value to support
	// efficient isDeployed queries
	if err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&v1beta1.Deployment{},
		appRefFilter,
		func(rawObj client.Object) []string {
			d := rawObj.(*v1beta1.Deployment)
			return []string{utils.GetAppRef(d)}
		}); err != nil {
		return err
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.Deployment{}).
		WithDefaulter(webhook).
		WithValidator(webhook).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-app-edge-orchestrator-intel-com-v1beta1-deployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=app.edge-orchestrator.intel.com,resources=deployments,verbs=create;update,versions=v1beta1,name=mdeployment.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-app-edge-orchestrator-intel-com-v1beta1-deployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=app.edge-orchestrator.intel.com,resources=deployments,verbs=create;update,versions=v1beta1,name=vdeployment.kb.io,admissionReviewVersions=v1

// Deployment implements a validating and defaulting webhook for Deployment.
type Deployment struct {
	Client          client.Reader
	catalogclient   catalogclient.CatalogClient
	vaultAuthClient auth.VaultAuth
	deleteGitRepo   bool
}

var _ webhook.CustomDefaulter = &Deployment{}
var _ webhook.CustomValidator = &Deployment{}

// Default adds catalog finalizer to the Deployment resource upon creation and update
func (webhook *Deployment) Default(ctx context.Context, obj runtime.Object) error {
	log := ctrl.LoggerFrom(ctx)

	d, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Deployment but got a %T", obj))
	}

	// Note that the finalizer needs to set not just create but also update because
	// PUT request from adm-gateway removes the existing finalizers
	if d.ObjectMeta.DeletionTimestamp.IsZero() && webhook.deleteGitRepo &&
		!cutil.ContainsFinalizer(d, v1beta1.FinalizerGitRemote) {
		log.V(1).Info("Deployment Webhook", "message", "Set gitremote finalizer")
		cutil.AddFinalizer(d, v1beta1.FinalizerGitRemote)
	}

	if d.ObjectMeta.DeletionTimestamp.IsZero() && !cutil.ContainsFinalizer(d, v1beta1.FinalizerDependency) {
		log.V(1).Info("Deployment Webhook", "message", "Set dependency finalizer")
		cutil.AddFinalizer(d, v1beta1.FinalizerDependency)
	}

	isEqual := reflect.DeepEqual(v1beta1.DeploymentPackageRef{}, d.Spec.DeploymentPackageRef)
	if !(isEqual) &&
		d.ObjectMeta.DeletionTimestamp.IsZero() &&
		!cutil.ContainsFinalizer(d, v1beta1.FinalizerCatalog) {
		log.V(1).Info("Deployment Webhook", "message", "Set catalog finalizer")
		cutil.AddFinalizer(d, v1beta1.FinalizerCatalog)
	}

	return nil
}

// ValidateCreate ensures isDeployed flag is set true for a new Deployment
// It returns error if it fails to set the flag so that the Deployment is not created
func (webhook *Deployment) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx)

	d, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Deployment but got a %T", obj))
	}

	if reflect.DeepEqual(v1beta1.DeploymentPackageRef{}, d.Spec.DeploymentPackageRef) {
		return nil, nil
	}

	if err := utils.HandleIsDeployed(ctx, webhook.catalogclient, webhook.vaultAuthClient, d, true); err != nil {
		return nil, apierrors.NewInternalError(errors.Wrapf(err, "failed to set IsDeployed for Deployment %s", d.Name))
	}

	log.V(1).Info("Deployment Webhook", "event", "create", "message", "Set isDeployed", "deployment", d.Name,
		"appName", d.Spec.DeploymentPackageRef.Name,
		"appVersion", d.Spec.DeploymentPackageRef.Version)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (webhook *Deployment) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx)

	old, ok := oldObj.(*v1beta1.Deployment)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Deployment but got a %T", old))
	}

	newDeployment, ok := newObj.(*v1beta1.Deployment)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Deployment but got a %T", newDeployment))
	}

	if reflect.DeepEqual(v1beta1.DeploymentPackageRef{}, newDeployment.Spec.DeploymentPackageRef) {
		return nil, nil
	}

	// Ensure AppName is the same as before
	if err := webhook.validate(old, newDeployment); err != nil {
		return nil, err
	}

	// If AppVersion didn't change, pass
	if old.Spec.DeploymentPackageRef.Version == newDeployment.Spec.DeploymentPackageRef.Version {
		log.V(1).Info("Deployment Webhook", "event", "update", "message", "No AppVersion change, ignore", "deployment", newDeployment.Name)
		return nil, nil
	}

	// Set isDeployed flag for the new AppVersion
	if err := utils.HandleIsDeployed(ctx, webhook.catalogclient, webhook.vaultAuthClient, newDeployment, true); err != nil {
		return nil, apierrors.NewInternalError(errors.Wrapf(err, "failed to set IsDeployed for %s while upgrade, "+
			"check Catalog service connection", newDeployment.Name))
	}

	log.V(1).Info("Deployment Webhook", "event", "update", "message", "Set isDeployed", "deployment", newDeployment.Name,
		"appName", newDeployment.Spec.DeploymentPackageRef.Name,
		"appVersion", newDeployment.Spec.DeploymentPackageRef.Version)

	// Unset isDeployed flag for the old AppVersion if there are no other
	// Deployments associated with the old AppVersion
	isDeployed, err := utils.IsDeployed(ctx, webhook.Client, old)
	if err != nil {
		return nil, apierrors.NewInternalError(errors.Wrapf(err, "failed to unset IsDeployed for %s", old.Name))
	}

	if !isDeployed {
		if err := utils.HandleIsDeployed(ctx, webhook.catalogclient, webhook.vaultAuthClient, old, false); err != nil {
			// TODO: Unset the new version as the upgrade failed
			// It is likely to fail unsetting the new version also fails and it requires
			// manual sync in this case
			return nil, apierrors.NewInternalError(errors.Wrapf(err, "failed to unset IsDeployed for %s while upgrade, "+
				"check Catalog service connection", old.Name))
		}
		log.V(1).Info("Deployment Webhook", "event", "update", "message", "Unset isDeployed", "deployment", old.Name,
			"appName", old.Spec.DeploymentPackageRef.Name,
			"appVersion", old.Spec.DeploymentPackageRef.Version)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
// This function is not expected to bel called
func (webhook *Deployment) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (webhook *Deployment) validate(old *v1beta1.Deployment, deployment *v1beta1.Deployment) error {
	var allErrs field.ErrorList

	// Return error if AppName was updated as ADM cannot handle this spec update
	specPath := field.NewPath("spec")
	if old.Spec.DeploymentPackageRef.Name != deployment.Spec.DeploymentPackageRef.Name {
		allErrs = append(allErrs, field.Invalid(specPath.Child("appName"),
			deployment.Spec.DeploymentPackageRef.Name,
			"must match previous spec.appName",
		))
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1beta1.GroupVersion.WithKind("Deployment").GroupKind(), deployment.Name, allErrs)
	}

	return nil
}
