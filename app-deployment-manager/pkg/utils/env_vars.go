// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// default values for git client
	defaultServiceAccount            = "orch-svc"
	defaultMount                     = "secret"
	defaultGitServicePath            = "ma_git_service"
	defaultGitServiceUsernameKey     = "username"
	defaultGitServicePasswordKey     = "password"
	defaultAWSServicePath            = "ma_aws_service"
	defaultAWSServiceRegionKey       = "region"
	defaultAWSServiceAccessKeyID     = "accessKeyID"
	defaultAWSServiceSecretAccessKey = "secretAccessKey"
	defaultAWSSSHKeyID               = "sshKeyID"
	defaultHarborServicePath         = "ma_harbor_service"
	defaultHarborServiceUsernameKey  = "username"
	defaultHarborServicePasswordKey  = "password"
	defaultHarborServiceCertKey      = "cacerts"
	defaultGitProxy                  = ""
	defaultGitCaCert                 = ""
	defaultSecretServiceEndpoint     = "http://vault.orch-platform.svc.cluster.local:8200" // #nosec G101

	// environment keys
	// for Git access - this is just key to get os environment
	envKeyGitUser            = "GIT_USER"
	envKeyGitPassword        = "GIT_PASSWORD"
	envKeyGitServer          = "GIT_SERVER"
	envKeyGitProvider        = "GIT_PROVIDER"
	envKeyGitProxy           = "GIT_PROXY"
	envKeyGitCaCert          = "GIT_CA_CERT"
	envKeyGitRegion          = "GIT_REGION"
	envKeyGitAccessKey       = "GIT_ACCESSKEY"
	envKeyGitSecretAccessKey = "GIT_SECRET_ACCESSKEY" // #nosec G101
	envKeyGitAwsSSHKey       = "GIT_AWSSSHKEY"

	// for secret service - this is just key to get os environment
	envKeyServiceAccount                              = "SERVICE_ACCOUNT"                           // #nosec G101
	envKeySecretServiceEnabled                        = "SECRET_SERVICE_ENABLED"                    // #nosec G101
	envKeySecretServiceEndpoint                       = "SECRET_SERVICE_ENDPOINT"                   // #nosec G101
	envKeySecretServiceMount                          = "SECRET_SERVICE_MOUNT"                      // #nosec G101
	envKeySecretServiceGitServicePath                 = "SECRET_GIT_SERVICE_PATH"                   // #nosec G101
	envKeySecretServiceGitServiceKVKeyUsername        = "SECRET_GIT_SERVICE_USERNAME_KVKEY"         // #nosec G101
	envKeySecretServiceGitServiceKVKeyPassword        = "SECRET_GIT_SERVICE_PASSWORD_KVKEY"         // #nosec G101
	envKeySecretServiceAWSServicePath                 = "SECRET_AWS_SERVICE_PATH"                   // #nosec G101
	envKeySecretServiceAWSServiceKVKeyRegion          = "SECRET_AWS_SERVICE_REGION_KVKEY"           // #nosec G101
	envKeySecretServiceAWSServiceKVKeyAccessKey       = "SECRET_AWS_SERVICE_ACCESSKEY_KVKEY"        // #nosec G101
	envKeySecretServiceAWSServiceKVKeySecretAccessKey = "SECRET_AWS_SERVICE_SECRET_ACCESSKEY_KVKEY" // #nosec G101
	envKeySecretServiceAWSServiceKVKeySSHKey          = "SECRET_AWS_SERVICE_SECRET_SSHKEY_KVKEY"    // #nosec G101
	envKeySecretServiceHarborServicePath              = "SECRET_HARBOR_SERVICE_PATH"                // #nosec G101
	envKeySecretServiceHarborServiceKVKeyUsername     = "SECRET_HARBOR_SERVICE_USERNAME_KVKEY"      // #nosec G101
	envKeySecretServiceHarborServiceKVKeyPassword     = "SECRET_HARBOR_SERVICE_PASSWORD_KVKEY"      // #nosec G101
	envKeySecretServiceHarborServiceKVKeyCert         = "SECRET_HARBOR_SERVICE_CERT_KVKEY"          // #nosec G101

	envKeyKeycloakServiceEndpoint = "KEYCLOAK_SERVICE_ENDPOINT"

	defaultGitPollingInterval = 15
)

func GetFleetGitPollingInterval() (*metav1.Duration, error) {
	pollingIntervalStr, ok := os.LookupEnv("FLEET_GIT_POLLING_INTERVAL")
	if !ok {
		return nil, errors.NewNotFound("FLEET_GIT_POLLING_INTERVAL env var not set")
	}

	pollingIntervalInt, err := strconv.Atoi(pollingIntervalStr)
	if err != nil {
		pollingIntervalInt = defaultGitPollingInterval
		log.Warnw(fmt.Sprintf("using FLEET_GIT_POLLING_INTERVAL default %d secs", defaultGitPollingInterval))
	}

	pollingInterval := &metav1.Duration{
		Duration: time.Duration(pollingIntervalInt) * time.Second,
	}

	return pollingInterval, nil
}

// IsOPAEnabled checks if OPA deployment is enabled
func IsOPAEnabled() bool {
	enabled := os.Getenv("OPA_ENABLED")
	return enabled == "true"
}

// GetStatusRefreshInterval returns the ADM status refresh interval in seconds
func GetStatusRefreshInterval() int {
	i := os.Getenv("STATUS_REFRESH_INTERVAL")
	interval, err := strconv.Atoi(i)
	if err != nil {
		return 0
	}
	return interval
}

// GetMessageSizeLimit gets message size limit
func GetMessageSizeLimit() (int64, error) {
	// os.LookupEnv returns string
	msgSizeLimitStr, ok := os.LookupEnv("MSG_SIZE_LIMIT")
	if !ok {
		return 0, errors.NewNotFound("MSG_SIZE_LIMIT env var not set")
	}

	msgSizeLimit, err := strconv.ParseInt(msgSizeLimitStr, 10, 64)
	if err != nil {
		return 0, err
	}

	msgSizeLimitBytes := msgSizeLimit * 1024 * 1024
	if msgSizeLimitBytes < 0 {
		return 0, errors.NewInvalid("MSG_SIZE_LIMIT causes overflow")
	}

	return msgSizeLimitBytes, nil
}

// GetServiceAccount returns service account
func GetServiceAccount() string {
	sa, ok := os.LookupEnv(envKeyServiceAccount)
	if !ok {
		return defaultServiceAccount
	}

	return sa
}

// GetSecretServiceMount returns secret service mount
func GetSecretServiceMount() string {
	mount, ok := os.LookupEnv(envKeySecretServiceMount)
	if !ok {
		return defaultMount
	}
	return mount
}

// GetSecretServiceGitServicePath returns secret service Git service path
func GetSecretServiceGitServicePath() string {
	path, ok := os.LookupEnv(envKeySecretServiceGitServicePath)
	if !ok {
		return defaultGitServicePath
	}
	return path
}

// GetSecretServiceGitServiceKVKeyUsername returns KV key for git username from secret service
func GetSecretServiceGitServiceKVKeyUsername() string {
	username, ok := os.LookupEnv(envKeySecretServiceGitServiceKVKeyUsername)
	if !ok {
		return defaultGitServiceUsernameKey
	}
	return username
}

// GetSecretServiceGitServiceKVKeyPassword returns KV key for git password from secret service
func GetSecretServiceGitServiceKVKeyPassword() string {
	passwd, ok := os.LookupEnv(envKeySecretServiceGitServiceKVKeyPassword)
	if !ok {
		return defaultGitServicePasswordKey
	}
	return passwd
}

// GetSecretServiceAWSServicePath returns secret service AWS service path
func GetSecretServiceAWSServicePath() string {
	path, ok := os.LookupEnv(envKeySecretServiceAWSServicePath)
	if !ok {
		return defaultAWSServicePath
	}
	return path
}

// GetSecretServiceAWSServiceKVKeyRegion returns KV key for region from secret service
func GetSecretServiceAWSServiceKVKeyRegion() string {
	region, ok := os.LookupEnv(envKeySecretServiceAWSServiceKVKeyRegion)
	if !ok {
		return defaultAWSServiceRegionKey
	}
	return region
}

// GetSecretServiceAWSServiceKVKeyAccessKey returns KV key for access key from secret service
func GetSecretServiceAWSServiceKVKeyAccessKey() string {
	accessKey, ok := os.LookupEnv(envKeySecretServiceAWSServiceKVKeyAccessKey)
	if !ok {
		return defaultAWSServiceAccessKeyID
	}
	return accessKey
}

// GetSecretServiceAWSServiceKVKeySecretAccessKey returns KV key for secret access key from secret service
func GetSecretServiceAWSServiceKVKeySecretAccessKey() string {
	secretAccessKey, ok := os.LookupEnv(envKeySecretServiceAWSServiceKVKeySecretAccessKey)
	if !ok {
		return defaultAWSServiceSecretAccessKey
	}
	return secretAccessKey
}

// GetSecretServiceAWSServiceKVKeySSHKey returns KV key for ssh key from secret service
func GetSecretServiceAWSServiceKVKeySSHKey() string {
	sshKey, ok := os.LookupEnv(envKeySecretServiceAWSServiceKVKeySSHKey)
	if !ok {
		return defaultAWSSSHKeyID
	}
	return sshKey
}

// GetSecretServiceHarborServicePath returns secret service AWS service path
func GetSecretServiceHarborServicePath() string {
	path, ok := os.LookupEnv(envKeySecretServiceHarborServicePath)
	if !ok {
		return defaultHarborServicePath
	}
	return path
}

// GetSecretServiceHarborServiceKVKeyUsername returns KV key for region from secret service
func GetSecretServiceHarborServiceKVKeyUsername() string {
	region, ok := os.LookupEnv(envKeySecretServiceHarborServiceKVKeyUsername)
	if !ok {
		return defaultHarborServiceUsernameKey
	}
	return region
}

// GetSecretServiceHarborServiceKVKeyPassword returns KV key for region from secret service
func GetSecretServiceHarborServiceKVKeyPassword() string {
	region, ok := os.LookupEnv(envKeySecretServiceHarborServiceKVKeyPassword)
	if !ok {
		return defaultHarborServicePasswordKey
	}
	return region
}

// GetSecretServiceHarborServiceKVKeyCert returns KV key for region from secret service
func GetSecretServiceHarborServiceKVKeyCert() string {
	region, ok := os.LookupEnv(envKeySecretServiceHarborServiceKVKeyCert)
	if !ok {
		return defaultHarborServiceCertKey
	}
	return region
}

// IsSecretServiceEnabled returns true if SecretService is enabled; otherwise false
func IsSecretServiceEnabled() (bool, error) {
	flag, ok := os.LookupEnv(envKeySecretServiceEnabled)
	if !ok {
		return false, errors.NewNotFound("SECRET_SERVICE_ENABLED environment var is not set")
	}

	if flag == "false" {
		return false, nil
	}

	return true, nil
}

// GetGitUser returns env value for git user
func GetGitUser() (string, error) {
	user, ok := os.LookupEnv(envKeyGitUser)
	if !ok {
		return "", errors.NewNotFound("secret service is disabled but GIT_USER env var not set")
	}
	return user, nil
}

// GetGitPassword returns env value for git password
func GetGitPassword() (string, error) {
	pw, ok := os.LookupEnv(envKeyGitPassword)
	if !ok {
		return "", errors.NewNotFound("secret service is disabled but GIT_PASSWORD env var not set")
	}
	return pw, nil
}

// GetGitServer returns env value for git server
func GetGitServer() (string, error) {
	server, ok := os.LookupEnv(envKeyGitServer)
	if !ok {
		return "", errors.NewNotFound("GIT_SERVER env var not set")
	}
	return server, nil
}

// GetGitProvider returns env value for git provider type
func GetGitProvider() (string, error) {
	provider, ok := os.LookupEnv(envKeyGitProvider)
	if !ok {
		return "", errors.NewNotFound("GIT_PROVIDER env var not set")
	}
	return provider, nil
}

// GetGitProxy returns env value for git proxy
func GetGitProxy() string {
	proxy, ok := os.LookupEnv(envKeyGitProxy)
	if !ok {
		return defaultGitProxy
	}
	return proxy
}

// GetGitRegion returns env value for git region (AWS)
func GetGitRegion() (string, error) {
	region, ok := os.LookupEnv(envKeyGitRegion)
	if !ok {
		return "", errors.NewNotFound("secret service is disabled but GIT_REGION env var not set")
	}
	return region, nil
}

// GetGitAccessKey returns env value for git access key (AWS)
func GetGitAccessKey() (string, error) {
	key, ok := os.LookupEnv(envKeyGitAccessKey)
	if !ok {
		return "", errors.NewNotFound("secret service is disabled but GIT_ACCESSKEY env var not set")
	}
	return key, nil
}

// GetGitSecretAccessKey returns env value for git secret access key (AWS)
func GetGitSecretAccessKey() (string, error) {
	key, ok := os.LookupEnv(envKeyGitSecretAccessKey)
	if !ok {
		return "", errors.NewNotFound("GIT_SECRET_ACCESSKEY env var not set")
	}
	return key, nil
}

// GetAwsSSHKey returns env value for git ssh key id (AWS)
func GetAwsSSHKey() (string, error) {
	sshKey, ok := os.LookupEnv(envKeyGitAwsSSHKey)
	if !ok {
		return "", errors.NewNotFound("GIT_AWSSSHKEY env var not set")
	}
	return sshKey, nil
}

// GetSecretServiceEndpoint returns secret service endpoint
func GetSecretServiceEndpoint() string {
	endpoint, ok := os.LookupEnv(envKeySecretServiceEndpoint)
	if !ok {
		return defaultSecretServiceEndpoint
	}
	return endpoint
}

func GetCAPIEnabled() (string, error) {
	enable, ok := os.LookupEnv("CAPI_ENABLED")
	if !ok {
		return "", errors.NewNotFound("CAPI_ENABLED env var not set")
	}

	return enable, nil
}

func GetIntegerFromEnv(envvar string, defaultvalue int) (int, string) {
	var value int
	var message string

	intStr, ok := os.LookupEnv(envvar)
	if ok {
		intval, err := strconv.Atoi(intStr)
		if err != nil {
			message = fmt.Sprintf("parsing %s failed - %s - using default of %d", envvar, err, defaultvalue)
			value = defaultvalue
		} else {
			value = intval
		}
	} else {
		message = fmt.Sprintf("failed to get %s from env - using default of %d", envvar, defaultvalue)
		value = defaultvalue
	}
	return value, message
}

// GetKeycloakServiceEndpoint returns secret service endpoint
func GetKeycloakServiceEndpoint() string {
	endpoint, ok := os.LookupEnv(envKeyKeycloakServiceEndpoint)
	if !ok {
		return ""
	}
	return endpoint
}
