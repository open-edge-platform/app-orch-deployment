// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	APIExtensionFinalizer  = "apiextension.app.edge-orchestrator.intel.com"
	APIExtensionAnnotation = "apiextension.app.edge-orchestrator.intel.com/name"

	IngressReady   APIExtensionConditionType = "IngressReady"
	TokenReady     APIExtensionConditionType = "TokenReady"
	AgentRepoReady APIExtensionConditionType = "AgentRepoReady"
	InitAgentReady APIExtensionConditionType = "InitAgentReady"
	AgentReady     APIExtensionConditionType = "AgentReady"
)

type APIExtensionConditionType string

// APIGroup indicates API group
type APIGroup struct {
	// Name defines the name of the API connection to be extended
	// It is used to generate the path in the external URL for extended API
	// access, so must be unique
	Name string `json:"name"`

	// Version defines the API group version
	Version string `json:"version"`
}

func (a *APIGroup) GetAPIGroupVersionPath() string {
	return fmt.Sprintf("%s/%s", a.Name, a.Version)
}

// ProxyEndpoint has endpoint information for proxy
type ProxyEndpoint struct {
	// ServiceName defines the name of the service to proxy
	ServiceName string `json:"serviceName"`

	// Path defines the path that follows the API group name and version in the
	// external URL to identify the backend service
	Path string `json:"path"`

	// Backend defines the backend service URL that API Proxy and Agent to
	// interacts with
	Backend string `json:"backend"`

	// Scheme defines the protocol scheme (http or https) for the backend service
	Scheme string `json:"scheme"`

	// AuthType defines the auth protocol (tls, mtls or insecure) for the backend service
	AuthType string `json:"authType"`

	// AppName indicates the target application name
	AppName string `json:"appName"`
}

// UIExtension is the schema for UI extension
type UIExtension struct {
	// ServiceName corresponds to ServiceName in ProxyEndpoint and refers to the name of the service to proxy.
	ServiceName string `json:"serviceName"`

	// Description states the purpose of the dashboard that this UIExtension represents.
	Description string `json:"description"`

	// Label represents a dashboard in the main UI.
	Label string `json:"label"`

	// The name of the main file to load this specific UI extension.
	FileName string `json:"fileName"`

	// The name of the application corresponding to this UI extension.
	AppName string `json:"appName"`

	// The application module to be loaded.
	ModuleName string `json:"moduleName"`
}

type APIAgentConfig struct {
	AgentId  string `yaml:"agentId"`
	ProxyURL string `yaml:"proxyURL"`
}

// TokenSecretRef is a reference of token secret
type TokenSecretRef struct {
	// Name of the secret used to store an auth token
	Name string `json:"name,omitempty"`

	// GeneratedToken is used to recreate the secret in case
	// when the secret is deleted.
	// TODO: Improve this with Secret Service later
	GeneratedToken string `json:"token,omitempty"`

	// Timestamp when token was generated
	Timestamp string `json:"timestamp,omitempty"`
}

func (t *TokenSecretRef) IsValid() bool {
	return len(t.Name) != 0 && len(t.GeneratedToken) != 0
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=apiextensions,shortName=apie
//+kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName",description="Display name of the API extension"
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.display`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
//+kubebuilder:printcolumn:name="Endpoints",type="string",JSONPath=".status.appliedApiEndpoints",description="Applied API endpoint URLs"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of this resource"
//+kubebuilder:storageversion

// APIExtension is the Schema for the apiextensions API
type APIExtension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIExtensionSpec   `json:"spec,omitempty"`
	Status APIExtensionStatus `json:"status,omitempty"`
}

// APIExtensionSpec defines the desired state of APIExtension
// GUI is optional for API extension
type APIExtensionSpec struct {
	// DisplayName is the display name of this API extension
	DisplayName string `json:"displayName"`

	// Project refers to the owner project of this API extension
	Project string `json:"project"`

	// APIGroup defines the API collection to be extended
	APIGroup APIGroup `json:"apiGroup"`

	// ProxyEndpoints defines proxy and target endpoints required to configure
	// API gateway and API Proxy Service to route extended API calls to the
	// backend services
	ProxyEndpoints []ProxyEndpoint `json:"proxyEndpoints"`

	// UIExtensions defines UI Extensions for API Extension
	UIExtensions []UIExtension `json:"uiExtensions,omitempty"`

	// AgentTargetClusters define the cluster labels for selecting clusters
	// to deploy API Agent service
	AgentClusterLabels map[string]string `json:"agentClusterLabels"`
}

// APIExtensionStatus defines the observed state of APIExtension
type APIExtensionStatus struct {
	// Conditions is a list conditions that describe the state of the extension CR
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// A string to display in the CLI
	Display string `json:"display,omitempty"`

	// The state of the API Extension (Running / Down)
	State StateType `json:"state"`

	// An informative error message if State is Down
	Message string `json:"message,omitempty"`

	// A list of applied API endpoints
	AppliedAPIEndpoints []string `json:"appliedApiEndpoints,omitempty"`

	// A reference of Token secret
	TokenSecretRef TokenSecretRef `json:"tokenSecretRef,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true

// APIExtensionList contains a list of APIExtension
type APIExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIExtension{}, &APIExtensionList{})
}
