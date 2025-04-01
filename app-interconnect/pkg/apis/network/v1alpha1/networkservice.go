// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NetworkClusterServiceKind = "NetworkService"
)

// NetworkServiceSpec is the spec for a NetworkService resource
type NetworkServiceSpec struct {
	NetworkRef  corev1.ObjectReference `json:"networkRef,omitempty"`
	ClusterRef  corev1.ObjectReference `json:"clusterRef,omitempty"`
	ServiceRef  corev1.ObjectReference `json:"serviceRef,omitempty"`
	ExposePorts []NetworkServicePort   `json:"exposePorts,omitempty"`
}

type NetworkServicePort struct {
	Port int32 `json:"port"`
}

type NetworkServiceStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Network",type=string,JSONPath=`.spec.networkRef.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef.name`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.serviceRef.namespace`
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.serviceRef.name`
// +genclient:nonNamespaced

// NetworkService is a custom resource for defining a NetworkService
type NetworkService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkServiceSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status NetworkServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// NetworkServiceList is a list of NetworkService resources
type NetworkServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkService `json:"items"`
}
