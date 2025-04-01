// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterKind = "Cluster"
)

// ClusterSpec is the spec for a Cluster resource
type ClusterSpec struct {
	ClusterRef corev1.ObjectReference `json:"clusterRef,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Configuring;Running;Terminating
type ClusterPhase string

const (
	ClusterPending     ClusterPhase = "Pending"
	ClusterConfiguring ClusterPhase = "Configuring"
	ClusterRunning     ClusterPhase = "Running"
	ClusterTerminating ClusterPhase = "Terminating"
)

// +kubebuilder:validation:Enum=None;LoadBalancer
type IngressType string

const (
	IngressNone         IngressType = "None"
	IngressLoadBalancer IngressType = "LoadBalancer"
)

type ClusterStatus struct {
	// +kubebuilder:default=true
	// +kubebuilder:default=Pending
	Phase ClusterPhase `json:"phase,omitempty"`
	// +kubebuilder:default=true
	// +kubebuilder:default=None
	Ingress  IngressType                   `json:"ingress,omitempty"`
	Services []corev1.LocalObjectReference `json:"services,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.clusterRef.namespace`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="IngressType",type=string,JSONPath=`.status.ingress`
// +genclient:nonNamespaced

// Cluster is a custom resource for defining a cluster
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// ClusterList is a list of Cluster resources
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}
