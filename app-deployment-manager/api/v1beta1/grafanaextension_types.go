// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	GrafanaExtensionFolderAnnotationKey = "grafana_folder"
	GrafanaExtensionDefaultFolder       = "AppOrch"
	GrafanaExtensionLabelDashboardKey   = "grafana_dashboard"
	GrafanaExtensionLabelDashboard      = "customer"
	GrafanaExtensionLabelProjectKey     = "project"

	GrafanaExtensionConditionConfigMapReady = "ConfigMapReady"
	GrafanaExtensionConditionDashboardReady = "DashboardReady"
	GrafanaExtensionConditionTerminating    = "Terminating"
)

// ArtifactRef contains artifact information
type ArtifactRef struct {
	// Publisher of the artifact
	Publisher string `json:"publisher"`

	// Name of the artifact
	Name string `json:"name"`

	// Description contains the detailed information of Grafana extension
	Description string `json:"description,omitempty"`

	// Artifact has the artifact contents, e.g., Grafana dashboard JSON model
	Artifact string `json:"artifact"`
}

// GrafanaExtensionSpec defines the desired state of GrafanaExtension
type GrafanaExtensionSpec struct {
	// DisplayName is the display name of this Grafana extension
	DisplayName string `json:"displayName"`

	// Project refers to the owner project of this Grafana extension
	Project string `json:"project"`

	// ArtifactRef contains artifact information
	ArtifactRef ArtifactRef `json:"artifactRef"`
}

// GrafanaExtensionStatus defines the observed state of GrafanaExtension
type GrafanaExtensionStatus struct {
	// Conditions is a list conditions that describe the state of the Grafana extension CR
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// A string to display in the CLI
	Display string `json:"display,omitempty"`

	// The state of the Grafana extension (Running / Down / Terminating)
	State StateType `json:"state"`

	// An informative error message if State is Down
	Message string `json:"message,omitempty"`

	// Time of last status update for the Grafana extension
	LastStatusUpdate string `json:"lastStatusUpdate,omitempty"`

	// The last generation that has been successfully reconciled
	ReconciledGeneration int64 `json:"reconciledGeneration,omitempty"`
}

func (g *GrafanaExtensionStatus) SetStatus(timestamp time.Time, state StateType, message, conditionType string, conditionStatus metav1.ConditionStatus, lastGen int64) {
	g.Display = fmt.Sprintf("%s:%s(%s)", state, conditionType, conditionStatus)
	g.State = state
	g.Message = message
	g.LastStatusUpdate = timestamp.Format(time.RFC3339)
	g.ReconciledGeneration = lastGen

	if len(g.Conditions) == 0 {
		g.Conditions = make([]metav1.Condition, 0)
	}

	hasCondition := false
	for i := 0; i < len(g.Conditions); i++ {
		// condition exists
		if g.Conditions[i].Type == conditionType {
			g.Conditions[i].Status = conditionStatus
			g.Conditions[i].LastTransitionTime.Time = timestamp
			g.Conditions[i].Reason = fmt.Sprintf("%s:%s", conditionType, conditionStatus)
			g.Conditions[i].Message = message
			hasCondition = true
		}
	}

	// new condition added
	if !hasCondition {
		g.Conditions = append(g.Conditions, metav1.Condition{
			Type:   conditionType,
			Status: conditionStatus,
			LastTransitionTime: metav1.Time{
				Time: timestamp,
			},
			Reason:  fmt.Sprintf("%s:%s", conditionType, conditionStatus),
			Message: message,
		})
	}
}

func (g *GrafanaExtensionStatus) GetCondition(conditionType string) metav1.Condition {
	for _, c := range g.Conditions {
		if c.Type == conditionType {
			return c
		}
	}

	return metav1.Condition{}
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=grafanaextensions,shortName=grafanaext
//+kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName",description="Display name of Grafana extension"
//+kubebuilder:printcolumn:name="Artifact Name",type="string",JSONPath=".spec.artifactRef.name",description="Artifact name"
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.display`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:printcolumn:name="Description",type="string",JSONPath=".spec.artifactRef.description",description="Description of Grafana extension"
// +kubebuilder:storageversion

// GrafanaExtension is the Schema for the grafanaextensions API
type GrafanaExtension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GrafanaExtensionSpec   `json:"spec,omitempty"`
	Status GrafanaExtensionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GrafanaExtensionList contains a list of GrafanaExtension
type GrafanaExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GrafanaExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GrafanaExtension{}, &GrafanaExtensionList{})
}
