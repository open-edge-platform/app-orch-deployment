// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NetworkKind = "Network"
)

// NetworkSpec is the spec for a Network resource
type NetworkSpec struct {
	// +optional
	NetworkRef *corev1.ObjectReference `json:"networkRef,omitempty"`
}

type NetworkStatus struct {
	DeploymentRefs []corev1.ObjectReference `json:"deploymentRefs,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.networkRef.namespace`
// +kubebuilder:printcolumn:name="Network",type=string,JSONPath=`.spec.networkRef.name`
// +genclient:nonNamespaced

// Network is a custom resource for defining a Network
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status NetworkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// NetworkList is a list of Network resources
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Network `json:"items"`
}
