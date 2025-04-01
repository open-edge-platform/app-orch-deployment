// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterLinkKind = "Link"
)

// LinkSpec is the spec for a Link resource
type LinkSpec struct {
	SourceClusterRef corev1.ObjectReference `json:"sourceClusterRef,omitempty"`
	TargetClusterRef corev1.ObjectReference `json:"targetClusterRef,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Linking;Linked;Unlinking
type LinkPhase string

const (
	LinkPending   LinkPhase = "Pending"
	LinkLinking   LinkPhase = "Linking"
	LinkLinked    LinkPhase = "Linked"
	LinkUnlinking LinkPhase = "Unlinking"
)

type LinkStatus struct {
	// +kubebuilder:default=true
	// +kubebuilder:default=Pending
	Phase LinkPhase `json:"phase,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.spec.sourceClusterRef.name`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetClusterRef.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +genclient:nonNamespaced

// Link is a custom resource for defining a Link
type Link struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LinkSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={}
	Status LinkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true

// LinkList is a list of Link resources
type LinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Link `json:"items"`
}
