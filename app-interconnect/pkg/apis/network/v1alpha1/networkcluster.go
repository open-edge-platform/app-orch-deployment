// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NetworkClusterKind = "NetworkCluster"
)

// NetworkClusterSpec is the spec for a NetworkCluster resource
type NetworkClusterSpec struct {
	NetworkRef corev1.ObjectReference `json:"networkRef,omitempty"`
	ClusterRef corev1.ObjectReference `json:"clusterRef,omitempty"`
}

// +kubebuilder:validation:Enum=Unknown;Spoke;Hub
type NetworkClusterRole string

const (
	NetworkClusterRoleUnknown NetworkClusterRole = "Unknown"
	NetworkClusterRoleSpoke   NetworkClusterRole = "Spoke"
	NetworkClusterRoleHub     NetworkClusterRole = "Hub"
)

type NetworkClusterStatus struct {
	// +kubebuilder:default=true
	// +kubebuilder:default=Unknown
	Role                  NetworkClusterRole            `json:"role,omitempty"`
	DeploymentClusterRefs []corev1.ObjectReference      `json:"deploymentClusterRefs,omitempty"`
	Services              []corev1.LocalObjectReference `json:"services,omitempty"`
	Links                 []corev1.LocalObjectReference `json:"links,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Network",type=string,JSONPath=`.spec.networkRef.name`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.clusterRef.namespace`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.status.role`
// +genclient:nonNamespaced

// NetworkCluster is a custom resource for defining a NetworkCluster
type NetworkCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkClusterSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status NetworkClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// NetworkClusterList is a list of NetworkCluster resources
type NetworkClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkCluster `json:"items"`
}
