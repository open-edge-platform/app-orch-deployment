// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CountType string

const (
	NotUsed       CountType = "notUsed"
	ClusterCounts CountType = "clusterCounts"
	AppCounts     CountType = "appCounts"
)

// Deployment status summary
type Summary struct {
	// Type of thing that we're counting
	Type CountType `json:"type"`

	// Sum over Status.Summary.DesiredReady from owned GitRepo objects
	Total int `json:"total,omitempty"`

	// Sum over Status.Summary.Running from owned GitRepo objects
	Running int `json:"running,omitempty"`

	// AppsTotal - AppsReady
	Down int `json:"down,omitempty"`

	// Unknown status to indicate cluster not reachable
	Unknown int `json:"unknown,omitempty"`
}

type Status struct {
	// The state of the object (Running, Down)
	State StateType `json:"state"`

	// An informative error message if object is Down
	Message string `json:"message,omitempty"`

	// Summary counts for objects below this one in the hiearchy
	Summary Summary `json:"summary,omitempty"`
}

type App struct {
	// Name of the app
	Name string `json:"name"`

	// Id of the app (equivalent to Fleet bundle name)
	Id string `json:"id"`

	// Last Deployment Generation applied by the Fleet agent
	DeploymentGeneration int64 `json:"deploymentGeneration"`

	// Status of the app
	Status Status `json:"status,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentClusterSpec defines the desired state of DeploymentCluster
// DeploymentClusters just exist to report status, so it is blank
type DeploymentClusterSpec struct {

	// DeploymentID is the ID of the corresponding Deployment
	DeploymentID string `json:"deploymentId"`

	// ClusterID is the ID of the corresponding cluster
	ClusterID string `json:"clusterId"`

	// Namespace is the Deployment / Cluster namespace
	Namespace string `json:"namespace"`
}

// DeploymentClusterStatus defines the observed state of DeploymentCluster
type DeploymentClusterStatus struct {
	// Conditions is a list conditions that describe the state of the deployment
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Name is the display name which user provides and ECM creates and assigns clustername label to Fleet cluster object
	Name string `json:"name"`

	// Status of the cluster
	Status Status `json:"status"`

	// A message summarizing the cluster status
	Display string `json:"display,omitempty"`

	// Last time status was updated
	LastStatusUpdate metav1.Time `json:"lastStatusUpdate"`

	// Per-app status on the cluster
	Apps []App `json:"apps,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.status.state`
//+kubebuilder:printcolumn:name="Apps-Ready",type=string,JSONPath=`.status.display`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.status.message`

// DeploymentCluster is the Schema for the deploymentclusters API
type DeploymentCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentClusterSpec   `json:"spec,omitempty"`
	Status DeploymentClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentClusterList contains a list of DeploymentCluster
type DeploymentClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentCluster{}, &DeploymentClusterList{})
}
