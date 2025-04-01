// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterServiceKind = "Service"
)

// ServiceSpec is the spec for a Service resource
type ServiceSpec struct {
	ClusterRef  corev1.ObjectReference `json:"clusterRef,omitempty"`
	ServiceRef  corev1.ObjectReference `json:"serviceRef,omitempty"`
	ExposePorts []ServicePort          `json:"exposePorts,omitempty"`
}

type ServicePort struct {
	Port int32 `json:"port"`
}

// +kubebuilder:validation:Enum=Pending;Exposing;Exposed;Unexposing
type ServicePhase string

const (
	ServicePending    ServicePhase = "Pending"
	ServiceExposing   ServicePhase = "Exposing"
	ServiceExposed    ServicePhase = "Exposed"
	ServiceUnexposing ServicePhase = "Unexposing"
)

type ServiceStatus struct {
	// +kubebuilder:default=true
	// +kubebuilder:default=Pending
	Phase ServicePhase `json:"phase,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.serviceRef.namespace`
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.serviceRef.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +genclient:nonNamespaced

// Service is a custom resource for defining a Service
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status ServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// ServiceList is a list of Service resources
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Service `json:"items"`
}
