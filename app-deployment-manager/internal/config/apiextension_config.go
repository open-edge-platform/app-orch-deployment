// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"sync"
)

type IngressKind string

const (
	Nginx   IngressKind = "nginx"
	Traefik IngressKind = "traefik"
)

type AuthType int

const (
	AuthTypeUnknown AuthType = iota
	AuthTypeInsecure
	AuthTypeMTLS
	AuthTypeTLS

	AuthTypeUnknownString  string = "unknown"
	AuthTypeInsecureString string = "insecure"
	AuthTypeMTLSString     string = "mtls"
	AuthTypeTLSString      string = "tls"
)

func (a AuthType) String() string {
	return [...]string{AuthTypeUnknownString, AuthTypeInsecureString, AuthTypeMTLSString, AuthTypeTLSString}[a]
}

type EndpointScheme int

const (
	EndpointSchemeUnknown EndpointScheme = iota
	EndpointSchemeHTTP
	EndpointSchemeHTTPS

	EndpointSchemeUnknownString string = "unknown"
	EndpointSchemeHTTPString    string = "http"
	EndpointSchemeHTTPSString   string = "https"

	EndpointSchemeUnknownPort int = 0
	EndpointSchemeHTTPPort    int = 80
	EndpointSchemeHTTPSPort   int = 443
)

func (s EndpointScheme) String() string {
	return [...]string{EndpointSchemeUnknownString, EndpointSchemeHTTPString, EndpointSchemeHTTPSString}[s]
}

func (s EndpointScheme) GetPort() int {
	return [...]int{EndpointSchemeUnknownPort, EndpointSchemeHTTPPort, EndpointSchemeHTTPSPort}[s]
}

var (
	apiExtensionConfig *APIExtensionConfig
	lock               sync.Mutex
)

type APIExtensionConfig struct {
	IngressKind          string `json:"ingressKind"`
	APIProxyURL          string `json:"apiProxyUrl"`
	APIProxyNamespace    string `json:"apiProxyNamespace,omitempty"`
	APIProxyService      string `json:"apiProxyServiceName"`
	APIProxyPort         int32  `json:"apiProxyServicePort"`
	APIGroupDomain       string `json:"apiGroupDomain"`
	APIAgentChartRepo    string `json:"apiAgentChartRepo"`
	APIAgentChart        string `json:"apiAgentChart"`
	APIAgentChartVersion string `json:"apiAgentChartVersion"`
	APIAgentNamespace    string `json:"apiAgentNamespace"`
	TokenExpiryDays      string `json:"tokenExpiryDays"`
}

func SetAPIExtensionConfig(cfg *APIExtensionConfig) error {
	lock.Lock()
	defer lock.Unlock()

	// TODO: validate inputs

	apiExtensionConfig = cfg
	return nil
}

func GetAPIExtensionConfig() *APIExtensionConfig {
	if apiExtensionConfig == nil {
		panic("config.Get() called before Set()")
	}
	return apiExtensionConfig
}
