// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StateType string
type DeploymentType string
type LabelType string

const (
	Down             StateType = "Down"
	Running          StateType = "Running"
	Unknown          StateType = "Unknown"
	Error            StateType = "Error"
	InternalError    StateType = "InternalError"
	Deploying        StateType = "Deploying"
	Updating         StateType = "Updating"
	Terminating      StateType = "Terminating"
	NoTargetClusters StateType = "NoTargetClusters"

	AutoScaling DeploymentType = "auto-scaling"
	Targeted    DeploymentType = "targeted"

	AppName               LabelType = "app.edge-orchestrator.intel.com/app-name"
	BundleName            LabelType = "app.edge-orchestrator.intel.com/bundle-name"
	BundleType            LabelType = "app.edge-orchestrator.intel.com/bundle-type"
	DeploymentID          LabelType = "app.edge-orchestrator.intel.com/deployment-id"
	FleetClusterID        LabelType = "fleet.cattle.io/cluster"
	FleetClusterNamespace LabelType = "fleet.cattle.io/cluster-namespace"
	LabelAppName          LabelType = "app.edge-orchestrator.intel.com/app-name"
	LabelBundleName       LabelType = "app.edge-orchestrator.intel.com/bundle-name"
	LabelBundleType       LabelType = "app.edge-orchestrator.intel.com/bundle-type"
	DeploymentGeneration  LabelType = "deploymentGeneration"

	ClusterName   LabelType = "edge-orchestrator.intel.com/clustername"
	CapiInfraName LabelType = "edge-orchestrator.intel.com/capiInfraName"

	// fleet-global.yaml labels used for extension apps
	FleetProjectID LabelType = "edge-orchestrator.intel.com/project-id"
	FleetHostUuid  LabelType = "edge-orchestrator.intel.com/host-uuid"

	AppOrchActiveProjectID LabelType = "app.edge-orchestrator.intel.com/project-id"

	FinalizerGitRemote  = "app.edge-orchestrator.intel.com/git-remote"
	FinalizerCatalog    = "app.edge-orchestrator.intel.com/catalog"
	FinalizerDependency = "app.edge-orchestrator.intel.com/dependency"

	// Namespace label to configure fleet rs proxy registry
	FleetRsProxy LabelType = "app.edge-orchestrator.intel.com/fleet-rs-secret"

	// Fleet's ClientSecretName for gitrepo to use when git pull
	FleetGitSecretName = "fleet-gitrepo-cred"
)

type Namespace struct {
	// Namespace name
	Name string `json:"name"`

	// Namespace labels
	Labels map[string]string `json:"labels,omitempty"`

	// Namespace annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

type DeploymentPackageRef struct {
	// Name of the deployment package
	Name string `json:"name"`

	// Version of the deployment package
	Version string `json:"version"`

	// Profile to use for the base Helm values
	ProfileName string `json:"profileName,omitempty"`

	// ForbidsMultipleDeployments is the flag to indicate if this package allows duplicated deployment or not
	ForbidsMultipleDeployments bool `json:"forbidsMultipleDeployments,omitempty"`

	// Namespace resource to be created before any other resource. This allows
	// complex namespaces to be defined with predefined labels and annotations.
	Namespaces []Namespace `json:"namespaces,omitempty"`
}

type DependentDeploymentRef struct {
	DeploymentPackageRef DeploymentPackageRef `json:"deploymentPackageRef"`

	DeploymentName string `json:"deploymentName,omitempty"`
}

type HelmApp struct {
	// Chart can refer to any go-getter URL or OCI registry based helm chart
	// URL. If Repo is set below this field is the name of the chart to lookup.
	Chart string `json:"chart"`

	// Version of the chart or semver constraint of the chart to find
	Version string `json:"version"`

	// Repo is a http/https url to a helm repo to download the chart from
	Repo string `json:"repo,omitempty"`

	// RepoSecretName contains the auth secret for a private helm repository.
	// Valid only when Repo is provided.
	RepoSecretName string `json:"repoSecretName,omitempty"`

	// ImageRegistry is a http/https url to a image registry to download
	// application container images
	ImageRegistry string `json:"imageRegistry,omitempty"`

	// ImageRegistrySecretName contains the auth secret for the private image
	// registry. Valid only when ImageRegistry is provided.
	ImageRegistrySecretName string `json:"imageRegistrySecretName,omitempty"`
}

type IgnoreResource struct {
	// Name of the resource to ignore
	Name string `json:"name"`

	// K8S resource kind to ignore
	Kind string `json:"kind"`

	// K8S resource namespace
	Namespace string `json:"namespace,omitempty"`
}

type Application struct {
	// Name of this application
	Name string `json:"name"`

	// Verseion of the application
	Version string `json:"version"`

	// Namespace refer to the default namespace to be applied to any namespace
	// scoped application resources
	Namespace string `json:"namespace,omitempty"`

	// NamespaceLabels are labels that will be appended to the namespace. It
	// only adds the labels when the application is deployed and does not remove
	// them when the application is deleted.
	NamespaceLabels map[string]string `json:"namespaceLabels,omitempty"`

	// Targets refer to the clusters which will be deployed to
	// If it's manual deployment, cluster id is set
	Targets []map[string]string `json:"targets,omitempty"`

	// ProfileSecretName contains the profile contents
	ProfileSecretName string `json:"profileSecretName,omitempty"`

	// ValueSecretName contains the deployment time overriding values
	ValueSecretName string `json:"valueSecretName,omitempty"`

	// DependsOn refers to the of applications which must be ready before this
	// application can be deployed
	DependsOn []string `json:"dependsOn,omitempty"`

	// RedeployAfterUpdate, when true, causes removal of the existing deployment
	// before any upgrades
	RedeployAfterUpdate bool `json:"redeployAfterUpdate,omitempty"`

	// IgnoreResources is a list of k8s resource type to ignore. Any manual
	// changes to the ignored resources will not be detected or corrected
	// automatically.
	IgnoreResources []IgnoreResource `json:"ignoreResources,omitempty"`

	// HelmApp refer to the helm chart type application specification
	HelmApp *HelmApp `json:"helmApp,omitempty"`

	// DependentDeploymentPackages has dependent deployment packages, which indicates application-level dependency
	DependentDeploymentPackages map[string]DeploymentPackageRef `json:"dependentDeploymentPackages,omitempty"`

	// If this flag is set, the services part of the application that are annotated should be exposed to other clusters
	EnableServiceExport bool `json:"enableServiceExport,omitempty"`
}

// DeploymentSpec defines the desired state of Deployment
type DeploymentSpec struct {
	// DisplayName of this deployment
	DisplayName string `json:"displayName"`

	// Project refers to the owner project of this deployment
	Project string `json:"project"`

	// DeploymentPackage information
	DeploymentPackageRef DeploymentPackageRef `json:"deploymentPackageRef"`

	// Applications is a list of applications included in this deployment
	Applications []Application `json:"applications"`

	// DeploymentType for this deployment, can be either auto-scaling or
	// targeted.
	DeploymentType DeploymentType `json:"deploymentType"`

	// ChildDeploymentList is the list of child deployment, which indicates deployment-level dependency
	ChildDeploymentList map[string]DependentDeploymentRef `json:"childDeploymentList,omitempty"`

	// NetworkRef a reference to Network Object for supporting interconnect between clusters
	NetworkRef corev1.ObjectReference `json:"networkRef,omitempty"`
}

// Deployment status summary
type ClusterSummary struct {
	// Number of total target clusters for the deployment
	Total int `json:"total,omitempty"`

	// Number of running clusters
	Running int `json:"running,omitempty"`

	// Number of down clusters
	Down int `json:"down,omitempty"`

	// Number of unknown clusters
	Unknown int `json:"unknown,omitempty"`
}

// DeploymentStatus defines the observed state of Deployment
type DeploymentStatus struct {
	// Conditions is a list conditions that describe the state of the deployment
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// A string to display in the CLI
	Display string `json:"display,omitempty"`

	// The state of the Deployment (Running / Down)
	State StateType `json:"state"`

	// An informative error message if State is Down
	Message string `json:"message,omitempty"`

	// Summary of all cluster counts
	Summary ClusterSummary `json:"summary,omitempty"`

	// Is the Deployment currently being changed by Fleet
	DeployInProgress bool `json:"deployInProgress,omitempty"`

	// Time of last force resync of an app
	LastForceResync string `json:"lastForceResync,omitempty"`

	// Time of last status update for the Deployment
	LastStatusUpdate string `json:"lastStatusUpdate,omitempty"`

	// The last generation that has been successfully reconciled
	ReconciledGeneration int64 `json:"reconciledGeneration,omitempty"`

	// ParentDeploymentList is the list of parent deployment, which indicates deployment-level dependency
	ParentDeploymentList map[string]DependentDeploymentRef `json:"parentDeploymentList,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=deployments,shortName=dep
//+kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
//+kubebuilder:printcolumn:name="Pkg Name",type=string,JSONPath=`.spec.deploymentPackageRef.name`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.deploymentPackageRef.version`
//+kubebuilder:printcolumn:name="Profile",type=string,JSONPath=`.spec.deploymentPackageRef.profileName`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Status (T/R/D/U)",type=string,JSONPath=`.status.display`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:storageversion

// Deployment is the Schema for the deployments API
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// Use auto-generated UID as deployment ID
func (d *Deployment) GetId() string {
	return string(d.UID)
}

//+kubebuilder:object:root=true

// DeploymentList contains a list of Deployment
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
