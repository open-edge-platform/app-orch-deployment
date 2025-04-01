// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"fmt"
	"time"

	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterConditionKubeconfig = "KubeconfigReady"
	ClusterOrchKeyProjectID    = "edge-orchestrator.intel.com/project-id"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// Name is the name of this cluster
	Name string `json:"name"`

	// DisplayName is the display name of this cluster
	DisplayName string `json:"displayName,omitempty"`

	// KubeConfigSecretName contains the secret name for the kubeconfig data
	KubeConfigSecretName string `json:"kubeConfigSecretName"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// Conditions is a list conditions that describe the state of the cluster
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// A string to display in the CLI
	Display string `json:"display,omitempty"`

	// The state of the Cluster (Running / Down)
	State StateType `json:"state"`

	// An informative error message if State is Down
	Message string `json:"message,omitempty"`

	// Time of last status update for the Cluster CR
	LastStatusUpdate string `json:"lastStatusUpdate,omitempty"`

	// The last generation that has been successfully reconciled
	ReconciledGeneration int64 `json:"reconciledGeneration,omitempty"`

	// The last generation of Cluster CR in Fleet
	FleetClusterLastGeneration int64 `json:"fleetClusterLastGeneration,omitempty"`

	// The status reported from Fleet
	FleetStatus FleetStatus `json:"fleetStatus,omitempty"`
}

// FleetStatus defines the observed state of Fleet
// avoid to directly reuse fleet since fleet version/api keep changed, which effects entire process
type FleetStatus struct {
	BundleSummary    BundleSummary    `json:"bundleSummary,omitempty"`
	ResourceCounts   ResourceCounts   `json:"resourceCounts,omitempty"`
	FleetAgentStatus FleetAgentStatus `json:"fleetAgentStatus,omitempty"`
	ClusterDisplay   ClusterDisplay   `json:"clusterDisplay,omitempty"`
}

type BundleSummary struct {
	NotReady     string `json:"notReady"`
	WaitApplied  string `json:"waitApplied"`
	ErrApplied   string `json:"errApplied"`
	OutOfSync    string `json:"outOfSync"`
	Modified     string `json:"modified"`
	Ready        string `json:"ready"`
	Pending      string `json:"pending"`
	DesiredReady string `json:"desiredReady"`
}

type ResourceCounts struct {
	Ready        string `json:"ready"`
	DesiredReady string `json:"desiredReady"`
	WaitApplied  string `json:"waitApplied"`
	Modified     string `json:"modified"`
	Orphaned     string `json:"orphaned"`
	Missing      string `json:"missing"`
	Unknown      string `json:"unknown"`
	NotReady     string `json:"notReady"`
}

type FleetAgentStatus struct {
	LastSeen          metav1.Time `json:"lastSeen,omitempty"`
	Namespace         string      `json:"namespace,omitempty"`
	NonReadyNodes     string      `json:"nonReadyNodes,omitempty"`
	ReadyNodes        string      `json:"readyNodes,omitempty"`
	NonReadyNodeNames []string    `json:"nonReadyNodeNames,omitempty"`
	ReadyNodeNames    []string    `json:"readyNodeNames,omitempty"`
}

type ClusterDisplay struct {
	ReadyBundles string `json:"readyBundles"`
	ReadyNodes   string `json:"readyNodes"`
	SampleNode   string `json:"sampleNode"`
	State        string `json:"state"`
}

func (c *ClusterStatus) SetStatus(timestamp time.Time, state StateType, message, conditionType string, conditionStatus metav1.ConditionStatus, lastGen int64, fcs fleetv1alpha1.ClusterStatus, fcGen int64) {
	c.Display = fmt.Sprintf("%s:%s(%s)", state, conditionType, conditionStatus)
	c.State = state
	c.Message = message
	c.LastStatusUpdate = timestamp.Format(time.RFC3339)
	c.ReconciledGeneration = lastGen
	c.FleetClusterLastGeneration = fcGen

	if len(c.Conditions) == 0 {
		c.Conditions = make([]metav1.Condition, 0)
	}

	hasCondition := false
	for i := 0; i < len(c.Conditions); i++ {
		// condition exists
		if c.Conditions[i].Type == conditionType {
			c.Conditions[i].Status = conditionStatus
			c.Conditions[i].LastTransitionTime.Time = timestamp
			c.Conditions[i].Reason = fmt.Sprintf("%s:%s", conditionType, conditionStatus)
			c.Conditions[i].Message = message
			hasCondition = true
		}
	}

	// new condition added
	if !hasCondition {
		c.Conditions = append(c.Conditions, metav1.Condition{
			Type:   conditionType,
			Status: conditionStatus,
			LastTransitionTime: metav1.Time{
				Time: timestamp,
			},
			Reason:  fmt.Sprintf("%s:%s", conditionType, conditionStatus),
			Message: message,
		})
	}

	c.FleetStatus.BundleSummary = c.deepCopyBundleSummary(fcs.Summary)
	c.FleetStatus.ResourceCounts = c.deepCopyResourceCounts(fcs.ResourceCounts)
	c.FleetStatus.FleetAgentStatus = c.deepCopyAgentStatus(fcs.Agent)
	c.FleetStatus.ClusterDisplay = c.deepCopyClusterDisplay(fcs.Display)

}

func (c *ClusterStatus) deepCopyBundleSummary(src fleetv1alpha1.BundleSummary) BundleSummary {
	return BundleSummary{
		NotReady:     fmt.Sprintf("%d", src.NotReady),
		WaitApplied:  fmt.Sprintf("%d", src.WaitApplied),
		ErrApplied:   fmt.Sprintf("%d", src.ErrApplied),
		OutOfSync:    fmt.Sprintf("%d", src.OutOfSync),
		Modified:     fmt.Sprintf("%d", src.Modified),
		Ready:        fmt.Sprintf("%d", src.Ready),
		Pending:      fmt.Sprintf("%d", src.Pending),
		DesiredReady: fmt.Sprintf("%d", src.DesiredReady),
	}
}

func (c *ClusterStatus) deepCopyResourceCounts(src fleetv1alpha1.GitRepoResourceCounts) ResourceCounts {
	return ResourceCounts{
		Ready:        fmt.Sprintf("%d", src.Ready),
		DesiredReady: fmt.Sprintf("%d", src.DesiredReady),
		WaitApplied:  fmt.Sprintf("%d", src.WaitApplied),
		Modified:     fmt.Sprintf("%d", src.Modified),
		Orphaned:     fmt.Sprintf("%d", src.Orphaned),
		Missing:      fmt.Sprintf("%d", src.Missing),
		Unknown:      fmt.Sprintf("%d", src.Unknown),
		NotReady:     fmt.Sprintf("%d", src.NotReady),
	}
}

func (c *ClusterStatus) deepCopyAgentStatus(src fleetv1alpha1.AgentStatus) FleetAgentStatus {
	nonReadyNodeNames := make([]string, 0)
	nonReadyNodeNames = append(nonReadyNodeNames, src.NonReadyNodeNames...)
	readyNodeNames := make([]string, 0)
	readyNodeNames = append(readyNodeNames, src.ReadyNodeNames...)

	return FleetAgentStatus{
		LastSeen:          src.LastSeen,
		Namespace:         src.Namespace,
		NonReadyNodes:     fmt.Sprintf("%d", src.NonReadyNodes),
		ReadyNodes:        fmt.Sprintf("%d", src.ReadyNodes),
		NonReadyNodeNames: nonReadyNodeNames,
		ReadyNodeNames:    readyNodeNames,
	}
}

func (c *ClusterStatus) deepCopyClusterDisplay(src fleetv1alpha1.ClusterDisplay) ClusterDisplay {
	result := ClusterDisplay{
		ReadyBundles: src.ReadyBundles,
		ReadyNodes:   src.ReadyNodes,
		SampleNode:   src.SampleNode,
		State:        src.State,
	}

	if result.ReadyBundles == "" {
		result.ReadyBundles = "nil"
	}
	if result.ReadyNodes == "" {
		result.ReadyNodes = "nil"
	}
	if result.SampleNode == "" {
		result.SampleNode = "nil"
	}
	if result.State == "" {
		result.State = "nil"
	}

	return result
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName",description="Display name of the Cluster CR"
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.display`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:storageversion

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
