// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NetworkLinkKind = "NetworkLink"
)

// NetworkLinkSpec is the spec for a NetworkLink resource
type NetworkLinkSpec struct {
	NetworkRef       corev1.ObjectReference `json:"networkRef,omitempty"`
	SourceClusterRef corev1.ObjectReference `json:"sourceClusterRef,omitempty"`
	TargetClusterRef corev1.ObjectReference `json:"targetClusterRef,omitempty"`
}

type NetworkLinkStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Network",type=string,JSONPath=`.spec.networkRef.name`
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.spec.sourceClusterRef.name`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetClusterRef.name`
// +genclient:nonNamespaced

// NetworkLink is a custom resource for defining a NetworkLink
type NetworkLink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkLinkSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status NetworkLinkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// NetworkLinkList is a list of NetworkLink resources
type NetworkLinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkLink `json:"items"`
}
