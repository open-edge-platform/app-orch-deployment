// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Utils", func() {
	const (
		// default values for git client
		defaultServiceAccount            = "orch-svc"
		defaultMount                     = "secret"
		defaultGitServicePath            = "ma_git_service"
		defaultGitServiceUsernameKey     = "username"
		defaultGitServicePasswordKey     = "password"
		defaultAWSServicePath            = "ma_aws_service"
		defaultAWSServiceRegion          = "region"
		defaultAWSServiceAccessKeyID     = "accessKeyID"
		defaultAWSServiceSecretAccessKey = "secretAccessKey"
		defaultAWSSSHKeyID               = "sshKeyID"
		defaultHarborServicePath         = "ma_harbor_service"
		defaultSecretServiceEndpoint     = "http://vault.orch-platform.svc.cluster.local:8200" // #nosec G101
		defaultGitProxy                  = ""
		defaultGitCaCert                 = ""
	)

	Describe("Test GetFleetGitPollingInterval", func() {
		It("successfully get value of fleet git polling interval from env vars", func() {
			expected := 60
			os.Setenv("FLEET_GIT_POLLING_INTERVAL", "60")
			v, err := GetFleetGitPollingInterval()
			Expect(err).ToNot(HaveOccurred())
			Expect(v.Duration).To(Equal(time.Duration(expected) * time.Second))
		})

		It("failed due to FLEET_GIT_POLLING_INTERVAL env var not set", func() {
			os.Unsetenv("FLEET_GIT_POLLING_INTERVAL")
			_, err := GetFleetGitPollingInterval()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetMessageSizeLimit", func() {
		It("successfully get message size limit", func() {
			os.Setenv("MSG_SIZE_LIMIT", "1")
			msgSizeLimitBytes, err := GetMessageSizeLimit()
			Expect(err).ToNot(HaveOccurred())
			Expect(msgSizeLimitBytes).To(Equal(int64(1048576)))
		})

		It("failed due to env missing", func() {
			os.Unsetenv("MSG_SIZE_LIMIT")
			_, err := GetMessageSizeLimit()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetServiceAccount", func() {
		It("successfully get service account from env vars", func() {
			expected := "test"
			os.Setenv("SERVICE_ACCOUNT", expected)
			v := GetServiceAccount()
			Expect(v).To(Equal(expected))
		})

		It("get default service account from env vars due to env missing", func() {
			os.Unsetenv("SERVICE_ACCOUNT")
			v := GetServiceAccount()
			Expect(v).To(Equal(defaultServiceAccount))
		})
	})

	Describe("Test IsOPAEnabled", func() {
		It("successfully get OPA Enabled true from env vars", func() {
			expected := "true"
			os.Setenv("OPA_ENABLED", expected)
			v := IsOPAEnabled()
			Expect(v).To(Equal(true))
		})

		It("successfully get OPA Enabled false when env var is missing", func() {
			os.Unsetenv("OPA_ENABLED")
			v := IsOPAEnabled()
			Expect(v).To(Equal(false))
		})
	})

	Describe("Test GetSecretServiceMount", func() {
		It("successfully get secret service mount from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_SERVICE_MOUNT", expected)
			v := GetSecretServiceMount()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service mount from env vars due to env missing", func() {
			os.Unsetenv("SECRET_SERVICE_MOUNT")
			v := GetSecretServiceMount()
			Expect(v).To(Equal(defaultMount))
		})
	})

	Describe("Test GetSecretServiceGitServicePath", func() {
		It("successfully get secret service git service path from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_GIT_SERVICE_PATH", expected)
			v := GetSecretServiceGitServicePath()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service git service path from env vars due to env missing", func() {
			os.Unsetenv("SECRET_GIT_SERVICE_PATH")
			v := GetSecretServiceGitServicePath()
			Expect(v).To(Equal(defaultGitServicePath))
		})
	})

	Describe("Test GetSecretServiceGitServiceKVKeyUsername", func() {
		It("successfully get secret service git username key from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_GIT_SERVICE_USERNAME_KVKEY", expected)
			v := GetSecretServiceGitServiceKVKeyUsername()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service git username key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_GIT_SERVICE_USERNAME_KVKEY")
			v := GetSecretServiceGitServiceKVKeyUsername()
			Expect(v).To(Equal(defaultGitServiceUsernameKey))
		})
	})

	Describe("Test GetSecretServiceGitServiceKVKeyPassword", func() {
		It("successfully get secret service git password key from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_GIT_SERVICE_PASSWORD_KVKEY", expected)
			v := GetSecretServiceGitServiceKVKeyPassword()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service git password key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_GIT_SERVICE_PASSWORD_KVKEY")
			v := GetSecretServiceGitServiceKVKeyPassword()
			Expect(v).To(Equal(defaultGitServicePasswordKey))
		})
	})

	Describe("Test GetSecretServiceAWSServicePath", func() {
		It("successfully get secret service aws service path from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_AWS_SERVICE_PATH", expected)
			v := GetSecretServiceAWSServicePath()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service aws service path from env vars due to env missing", func() {
			os.Unsetenv("SECRET_AWS_SERVICE_PATH")
			v := GetSecretServiceAWSServicePath()
			Expect(v).To(Equal(defaultAWSServicePath))
		})
	})

	Describe("Test GetSecretServiceAWSServiceKVKeyRegion", func() {
		It("successfully get secret service aws region key from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_AWS_SERVICE_REGION_KVKEY", expected)
			v := GetSecretServiceAWSServiceKVKeyRegion()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service aws region key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_AWS_SERVICE_REGION_KVKEY")
			v := GetSecretServiceAWSServiceKVKeyRegion()
			Expect(v).To(Equal(defaultAWSServiceRegion))
		})
	})

	Describe("Test GetSecretServiceAWSServiceKVKeyAccessKey", func() {
		It("successfully get secret service aws accesskey key from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_AWS_SERVICE_ACCESSKEY_KVKEY", expected)
			v := GetSecretServiceAWSServiceKVKeyAccessKey()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service aws accesskey key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_AWS_SERVICE_ACCESSKEY_KVKEY")
			v := GetSecretServiceAWSServiceKVKeyAccessKey()
			Expect(v).To(Equal(defaultAWSServiceAccessKeyID))
		})
	})

	Describe("Test GetSecretServiceAWSServiceKVKeySecretAccessKey", func() {
		It("successfully get secret service aws secretaccesskey key from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_AWS_SERVICE_SECRET_ACCESSKEY_KVKEY", expected)
			v := GetSecretServiceAWSServiceKVKeySecretAccessKey()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service aws secretaccesskey key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_AWS_SERVICE_SECRET_ACCESSKEY_KVKEY")
			v := GetSecretServiceAWSServiceKVKeySecretAccessKey()
			Expect(v).To(Equal(defaultAWSServiceSecretAccessKey))
		})
	})

	Describe("Test GetSecretServiceAWSServiceKVKeySSHKey", func() {
		It("successfully get secret service aws ssh key from env vars", func() {
			expected := "test-ssh-key"
			os.Setenv("SECRET_AWS_SERVICE_SECRET_SSHKEY_KVKEY", expected)
			v := GetSecretServiceAWSServiceKVKeySSHKey()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service aws ssh key from env vars due to env missing", func() {
			os.Unsetenv("SECRET_AWS_SERVICE_SECRET_SSHKEY_KVKEY")
			v := GetSecretServiceAWSServiceKVKeySSHKey()
			Expect(v).To(Equal(defaultAWSSSHKeyID))
		})
	})

	Describe("Test GetSecretServiceHarborServicePath", func() {
		It("successfully get secret service Harbor path from env vars", func() {
			expected := "test-service-path"
			os.Setenv("SECRET_HARBOR_SERVICE_PATH", expected)
			v := GetSecretServiceHarborServicePath()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service Harbor path from env vars due to env missing", func() {
			os.Unsetenv("SECRET_HARBOR_SERVICE_PATH")
			v := GetSecretServiceHarborServicePath()
			Expect(v).To(Equal(defaultHarborServicePath))
		})
	})

	Describe("Test GetSecretServiceHarborServiceKVKeyUsername", func() {
		It("successfully get secret service Harbor username from env vars", func() {
			expected := "test-service-username"
			os.Setenv("SECRET_HARBOR_SERVICE_USERNAME_KVKEY", expected)
			v := GetSecretServiceHarborServiceKVKeyUsername()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service Harbor username from env vars due to env missing", func() {
			os.Unsetenv("SECRET_HARBOR_SERVICE_USERNAME_KVKEY")
			v := GetSecretServiceHarborServiceKVKeyUsername()
			Expect(v).To(Equal(defaultHarborServiceUsernameKey))
		})
	})

	Describe("Test GetSecretServiceHarborServiceKVKeyPassword", func() {
		It("successfully get secret service Harbor password from env vars", func() {
			expected := "test-service-password"
			os.Setenv("SECRET_HARBOR_SERVICE_PASSWORD_KVKEY", expected)
			v := GetSecretServiceHarborServiceKVKeyPassword()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service Harbor password from env vars due to env missing", func() {
			os.Unsetenv("SECRET_HARBOR_SERVICE_PASSWORD_KVKEY")
			v := GetSecretServiceHarborServiceKVKeyPassword()
			Expect(v).To(Equal(defaultHarborServicePasswordKey))
		})
	})

	Describe("Test GetSecretServiceHarborServiceKVKeyCert", func() {
		It("successfully get secret service Harbor cert from env vars", func() {
			expected := "test-service-cert"
			os.Setenv("SECRET_HARBOR_SERVICE_CERT_KVKEY", expected)
			v := GetSecretServiceHarborServiceKVKeyCert()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service Harbor cert from env vars due to env missing", func() {
			os.Unsetenv("SECRET_HARBOR_SERVICE_CERT_KVKEY")
			v := GetSecretServiceHarborServiceKVKeyCert()
			Expect(v).To(Equal(defaultHarborServiceCertKey))
		})
	})

	Describe("Test IsSecretServiceEnabled", func() {
		It("successfully get flag to check if secret service enabled or not from env vars", func() {
			expected := true
			os.Setenv("SECRET_SERVICE_ENABLED", "true")
			v, err := IsSecretServiceEnabled()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))

			expected = false
			os.Setenv("SECRET_SERVICE_ENABLED", "false")
			v, err = IsSecretServiceEnabled()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed to get flag to check if secret service enabled or not due to env missing", func() {
			os.Unsetenv("SECRET_SERVICE_ENABLED")
			v, err := IsSecretServiceEnabled()
			Expect(err).To(HaveOccurred())
			Expect(v).To(Equal(false))
		})
	})

	Describe("Test GetSecretServiceEndpoint", func() {
		It("successfully get secret service endpoint from env vars", func() {
			expected := "test"
			os.Setenv("SECRET_SERVICE_ENDPOINT", expected)
			v := GetSecretServiceEndpoint()
			Expect(v).To(Equal(expected))
		})

		It("get default secret service endpoint from env vars due to env missing", func() {
			os.Unsetenv("SECRET_SERVICE_ENDPOINT")
			v := GetSecretServiceEndpoint()
			Expect(v).To(Equal(defaultSecretServiceEndpoint))
		})
	})

	Describe("Test GetGitUser", func() {
		It("successfully get git user from env vars", func() {
			expected := "test"
			os.Setenv("GIT_USER", expected)
			v, err := GetGitUser()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git user due to env missing", func() {
			os.Unsetenv("GIT_USER")
			_, err := GetGitUser()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitPassword", func() {
		It("successfully get git password from env vars", func() {
			expected := "test"
			os.Setenv("GIT_PASSWORD", expected)
			v, err := GetGitPassword()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git password due to env missing", func() {
			os.Unsetenv("GIT_PASSWORD")
			_, err := GetGitPassword()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitServer", func() {
		It("successfully get git server from env vars", func() {
			expected := "test"
			os.Setenv("GIT_SERVER", expected)
			v, err := GetGitServer()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git server due to env missing", func() {
			os.Unsetenv("GIT_SERVER")
			_, err := GetGitServer()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitProvider", func() {
		It("successfully get git provider from env vars", func() {
			expected := "test"
			os.Setenv("GIT_PROVIDER", expected)
			v, err := GetGitProvider()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git provider due to env missing", func() {
			os.Unsetenv("GIT_PROVIDER")
			_, err := GetGitProvider()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitProxy", func() {
		It("successfully get git proxy from env vars", func() {
			expected := "test"
			os.Setenv("GIT_PROXY", expected)
			v := GetGitProxy()
			Expect(v).To(Equal(expected))
		})

		It("get default git proxy from env vars due to env missing", func() {
			os.Unsetenv("GIT_PROXY")
			v := GetGitProxy()
			Expect(v).To(Equal(defaultGitProxy))
		})
	})

	Describe("Test GetGitCACert", func() {
		It("get default git proxy from env vars due to env missing", func() {
			os.Unsetenv("GIT_CA_CERT")
			v := GetGitProxy()
			Expect(v).To(Equal(defaultGitCaCert))
		})
	})

	Describe("Test GetGitRegion", func() {
		It("successfully get git region (AWS) from env vars", func() {
			expected := "test"
			os.Setenv("GIT_REGION", expected)
			v, err := GetGitRegion()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git region (AWS) due to env missing", func() {
			os.Unsetenv("GIT_REGION")
			_, err := GetGitRegion()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitAccessKey", func() {
		It("successfully get git accesskey (AWS) from env vars", func() {
			expected := "test"
			os.Setenv("GIT_ACCESSKEY", expected)
			v, err := GetGitAccessKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git accesskey (AWS) due to env missing", func() {
			os.Unsetenv("GIT_ACCESSKEY")
			_, err := GetGitAccessKey()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetGitSecretAccessKey", func() {
		It("successfully get git secretaccesskey (AWS) from env vars", func() {
			expected := "test"
			os.Setenv("GIT_SECRET_ACCESSKEY", expected)
			v, err := GetGitSecretAccessKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git secretaccesskey (AWS) due to env missing", func() {
			os.Unsetenv("GIT_SECRET_ACCESSKEY")
			_, err := GetGitSecretAccessKey()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetAwsSSHKey", func() {
		It("successfully get git ssh key (AWS) from env vars", func() {
			expected := "test-ssh-key"
			os.Setenv("GIT_AWSSSHKEY", expected)
			v, err := GetAwsSSHKey()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(expected))
		})

		It("failed get git ssh key (AWS) due to env missing", func() {
			os.Unsetenv("GIT_AWSSSHKEY")
			_, err := GetAwsSSHKey()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test GetStatusRefreshInterval", func() {
		It("successfully got status refresh interval from env vars", func() {
			expected := 10
			os.Setenv("STATUS_REFRESH_INTERVAL", strconv.Itoa(expected))
			v := GetStatusRefreshInterval()
			Expect(v).To(Equal(expected))
		})

		It("got 0 value when env var missing", func() {
			os.Unsetenv("STATUS_REFRESH_INTERVAL")
			v := GetStatusRefreshInterval()
			Expect(v).To(Equal(0))
		})
	})

	Describe("Test GetIntegerFromEnv", func() {
		envvar := "TEST_INTEGER_IN_ENV"
		defaultvalue := 20
		It("successfully got integer from env vars", func() {
			expected := 10
			os.Setenv(envvar, strconv.Itoa(expected))
			v, m := GetIntegerFromEnv(envvar, defaultvalue)
			Expect(v).To(Equal(expected))
			Expect(m).To(BeEmpty())
		})

		It("got default value when env var missing", func() {
			os.Unsetenv(envvar)
			v, m := GetIntegerFromEnv(envvar, defaultvalue)
			Expect(v).To(Equal(defaultvalue))
			Expect(m).To(Equal("failed to get TEST_INTEGER_IN_ENV from env - using default of 20"))
		})

		It("got default value when env var cannot be parsed", func() {
			os.Setenv(envvar, "Not an integer!")
			v, m := GetIntegerFromEnv(envvar, defaultvalue)
			Expect(v).To(Equal(defaultvalue))
			Expect(m).To(Equal("parsing TEST_INTEGER_IN_ENV failed - strconv.Atoi: parsing \"Not an integer!\": invalid syntax - using default of 20"))
		})
	})
})
