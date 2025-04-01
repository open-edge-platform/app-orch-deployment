// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package apiextension

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"math/big"
	"os"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/genericcondition"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vault "github.com/hashicorp/vault/api"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	manager "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/vault"
)

// These Repository objects are used to test recovery from various errors
type mockRepository struct {
	ExistsOnRemoteValue         bool
	ExistsOnRemoteError         error
	InitializeError             error
	CloneError                  error
	CommitFilesError            error
	PushToRemoteError           error
	DeleteError                 error
	CreateCodeCommitClientError error
	CreateRepoError             error
	DeleteRepoError             error
}

func (m *mockRepository) ExistsOnRemote() (bool, error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] ExistsOnRemote returning %v, %v\n", m.ExistsOnRemoteValue, m.ExistsOnRemoteError)
	return m.ExistsOnRemoteValue, m.ExistsOnRemoteError
}
func (m *mockRepository) Initialize(basedir string) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] Initialize returning %v\n", m.InitializeError)
	return m.InitializeError
}
func (m *mockRepository) Clone(basedir string) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] Clone returning %v\n", m.CloneError)
	return m.CloneError
}
func (m *mockRepository) CommitFiles() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] CommitFiles returning %v\n", m.CommitFilesError)
	return m.CommitFilesError
}
func (m *mockRepository) PushToRemote() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] PushToRemote returning %v\n", m.PushToRemoteError)
	return m.PushToRemoteError
}
func (m *mockRepository) Delete() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] PushToRemote returning %v\n", m.DeleteError)
	return m.DeleteError
}

func (m *mockRepository) CreateCodeCommitClient() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] CreateCodeCommitClient returning %v\n", m.CreateCodeCommitClientError)
	return m.CreateCodeCommitClientError
}

func (m *mockRepository) CreateRepo() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] CreateRepo returning %v\n", m.CreateRepoError)
	return m.CreateRepoError
}

func (m *mockRepository) DeleteRepo() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] DeleteRepo returning %v\n", m.DeleteRepoError)
	return m.DeleteRepoError
}

var _ = Describe("API Extension Controller", func() {
	const (
		timeout  = time.Second * 100
		duration = time.Second * 10
		interval = time.Second * 1

		// Configurations
		apiGroupDomain       = "orchestrator-extension.apis"
		apiProxyService      = "api-proxy"
		apiProxyPort         = int32(8123)
		tlsHost              = "app-orch.com"
		tlsSecret            = "tls-orch"
		apiAgentChartRepo    = "https://fake-chart-repo"
		apiAgentChart        = "api-agent"
		apiAgentChartVersion = "0.1.0"
		apiAgentNamespace    = "orch-app"

		// Test API extension spec values
		apiExtensionName = "test-api-extension"
		apiGroupName     = "test"
		apiGroupVersion  = "v1"
		namespaceName    = "default"

		svc1Name     = "test-service-1"
		svc1Path     = "svc1"
		svc1Backend  = "svc1.default:8080"
		svc1Scheme   = "http"
		svc1AuthType = "insecure"

		svc2Name     = "test-service-2"
		svc2Path     = "svc2"
		svc2Backend  = "svc2.default:8080"
		svc2Scheme   = "http"
		svc2AuthType = "insecure"

		svc3Name     = "test-service-3"
		svc3Path     = "svc3"
		svc3Backend  = "svc3.default:8080"
		svc3Scheme   = "http"
		svc3AuthType = "insecure"

		svc1UIExtDescription = "svc1 description"
		svc1UIExtLabel       = "svc1 label"
		svc1UIExtFileName    = "svc1filename"
		svc1UIExtAppName     = "svc1appname"
		svc1UIExtModuleName  = "svc1modulename"

		svc2UIExtDescription = "svc1 description"
		svc2UIExtLabel       = "svc1 label"
		svc2UIExtFileName    = "svc1filename"
		svc2UIExtAppName     = "svc1appname"
		svc2UIExtModuleName  = "svc1modulename"

		nginxIngress      = "nginx"
		nginxAnnotation   = "nginx.ingress.kubernetes.io/rewrite-target"
		traefikIngress    = "traefik"
		traefikAnnotation = "traefik.ingress.kubernetes.io/rewrite-target"

		gitServer   = "http://fake-server"
		gitUser     = "fake-user"
		gitPass     = "fake-pass"
		gitProvider = "gitea"

		secretServiceEnabled = "false"
		helmSecretName       = "fake-helm-secret"
	)

	var (
		m    manager.NewManagerFunc
		kind = reflect.TypeOf(v1beta1.APIExtension{}).Name()
		gvk  = v1beta1.GroupVersion.WithKind(kind)

		// API extension definition for testing
		apiExtension          *v1beta1.APIExtension
		apiExtensionLookupKey = types.NamespacedName{
			Name:      apiExtensionName,
			Namespace: namespaceName,
		}

		secretLookupKey = types.NamespacedName{
			Name:      fmt.Sprintf("%s-token", apiExtensionName),
			Namespace: namespaceName,
		}

		// Fleet GitRepo for validation
		agentClusters    = map[string]string{"test-target-label": "test"}
		gitrepoLookupKey = types.NamespacedName{
			Name:      fmt.Sprintf("%s-api-agent", apiExtensionName),
			Namespace: namespaceName,
		}
		gitrepo = fleetv1alpha1.GitRepo{ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      fmt.Sprintf("%s-api-agent", apiExtensionName),
			Annotations: map[string]string{
				v1beta1.APIExtensionAnnotation: apiExtensionName,
			},
		},
			Spec: fleetv1alpha1.GitRepoSpec{
				Repo:  "valiidate-runtime",
				Paths: []string{"api-agent"},
				Targets: []fleetv1alpha1.GitTarget{
					{
						Name: "match", ClusterSelector: &metav1.LabelSelector{
							MatchLabels: agentClusters,
						},
					},
				},
			},
		}

		// Ingress rules for validation
		pathType         = networkv1.PathTypePrefix
		ingSvc1LookupKey = types.NamespacedName{Name: svc1Name, Namespace: namespaceName}
		ingSvc1Rules     = []networkv1.IngressRule{
			{
				IngressRuleValue: networkv1.IngressRuleValue{
					HTTP: &networkv1.HTTPIngressRuleValue{
						Paths: []networkv1.HTTPIngressPath{
							{
								PathType: &pathType,
								Path: fmt.Sprintf("/%s.%s/%s/%s(/|$)(.*)",
									apiGroupName,
									apiGroupDomain,
									apiGroupVersion,
									svc1Path),
								Backend: networkv1.IngressBackend{
									Service: &networkv1.IngressServiceBackend{
										Name: apiProxyService,
										Port: networkv1.ServiceBackendPort{
											Number: apiProxyPort,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		ingSvc1Annotation = map[string]string{
			v1beta1.APIExtensionAnnotation: apiExtensionName,
			nginxAnnotation:                fmt.Sprintf("/client/%s/%s/$2", apiExtensionName, svc1Backend),
		}
		//ingSvc1TraefikAnnotation = map[string]string{
		//	v1alpha1.APIExtensionAnnotation: apiExtensionName,
		//	traefikAnnotation:               fmt.Sprintf("/client/%s/%s/$2", apiExtensionName, svc1Backend),
		//}

		ingSvc2LookupKey = types.NamespacedName{Name: svc2Name, Namespace: namespaceName}
		ingSvc2Rules     = []networkv1.IngressRule{
			{
				IngressRuleValue: networkv1.IngressRuleValue{
					HTTP: &networkv1.HTTPIngressRuleValue{
						Paths: []networkv1.HTTPIngressPath{
							{
								PathType: &pathType,
								Path: fmt.Sprintf("/%s.%s/%s/%s(/|$)(.*)",
									apiGroupName,
									apiGroupDomain,
									apiGroupVersion,
									svc2Path),
								Backend: networkv1.IngressBackend{
									Service: &networkv1.IngressServiceBackend{
										Name: apiProxyService,
										Port: networkv1.ServiceBackendPort{
											Number: apiProxyPort,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		ingSvc2Annotation = map[string]string{
			v1beta1.APIExtensionAnnotation: apiExtensionName,
			nginxAnnotation:                fmt.Sprintf("/client/%s/%s/$2", apiExtensionName, svc2Backend),
		}
	)

	BeforeEach(func() {
		// Ensure controller configurations are set as we want
		config.GetAPIExtensionConfig().IngressKind = nginxIngress
		config.GetAPIExtensionConfig().APIGroupDomain = apiGroupDomain
		config.GetAPIExtensionConfig().APIProxyService = apiProxyService
		config.GetAPIExtensionConfig().APIProxyPort = apiProxyPort
		config.GetAPIExtensionConfig().APIAgentChartRepo = apiAgentChartRepo
		config.GetAPIExtensionConfig().APIAgentChart = apiAgentChart
		config.GetAPIExtensionConfig().APIAgentChartVersion = apiAgentChartVersion
		config.GetAPIExtensionConfig().APIAgentNamespace = apiAgentNamespace

		os.Setenv("GIT_SERVER", gitServer)
		os.Setenv("GIT_USER", gitUser)
		os.Setenv("GIT_PASSWORD", gitPass)
		os.Setenv("GIT_PROVIDER", gitProvider)
		os.Setenv("SECRET_SERVICE_ENABLED", secretServiceEnabled)
		os.Setenv("FLEET_GIT_REMOTE_TYPE", "https")
		os.Setenv("API_AGENT_HELM_SECRET_NAME", helmSecretName)

		// Ensure resources used in the test do not exist
		Expect(k8sClient.Get(ctx, apiExtensionLookupKey, &v1beta1.APIExtension{})).ShouldNot(Succeed())
		Expect(k8sClient.Get(ctx, secretLookupKey, &corev1.Secret{})).ShouldNot(Succeed())
		Expect(k8sClient.Get(ctx, gitrepoLookupKey, &fleetv1alpha1.GitRepo{})).ShouldNot(Succeed())
		Expect(k8sClient.Get(ctx, ingSvc1LookupKey, &networkv1.Ingress{})).ShouldNot(Succeed())
		Expect(k8sClient.Get(ctx, ingSvc2LookupKey, &networkv1.Ingress{})).ShouldNot(Succeed())

		// Define APIExtension CR that will be used in the test
		apiExtension = &v1beta1.APIExtension{ObjectMeta: metav1.ObjectMeta{
			Name:      apiExtensionName,
			Namespace: namespaceName,
		},
			Spec: v1beta1.APIExtensionSpec{
				APIGroup: v1beta1.APIGroup{
					Name:    apiGroupName,
					Version: apiGroupVersion,
				},
				ProxyEndpoints: []v1beta1.ProxyEndpoint{
					{
						ServiceName: svc1Name,
						Path:        svc1Path,
						Backend:     svc1Backend,
						Scheme:      svc1Scheme,
						AuthType:    svc1AuthType,
					},
					{
						ServiceName: svc2Name,
						Path:        svc2Path,
						Backend:     svc2Backend,
						Scheme:      svc2Scheme,
						AuthType:    svc2AuthType,
					},
					{
						ServiceName: svc3Name,
						Path:        svc3Path,
						Backend:     svc3Backend,
						Scheme:      svc3Scheme,
						AuthType:    svc3AuthType,
					},
				},
				AgentClusterLabels: agentClusters,
				UIExtensions: []v1beta1.UIExtension{
					{
						ServiceName: svc1Name,
						Description: svc1UIExtDescription,
						Label:       svc1UIExtLabel,
						FileName:    svc1UIExtFileName,
						AppName:     svc1UIExtAppName,
						ModuleName:  svc1UIExtModuleName,
					},
					{
						ServiceName: svc2Name,
						Description: svc2UIExtDescription,
						Label:       svc2UIExtLabel,
						FileName:    svc2UIExtFileName,
						AppName:     svc2UIExtAppName,
						ModuleName:  svc2UIExtModuleName,
					},
				},
			},
		}
		// Mock responses from the Secret Service
		m = manager.NewManager
		manager.NewManager = newMockManager
	})

	AfterEach(func() {
		manager.NewManager = m
		gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
			return &mockRepository{
				ExistsOnRemoteValue: false,
				ExistsOnRemoteError: nil,
				InitializeError:     nil,
				CloneError:          nil,
				CommitFilesError:    nil,
				PushToRemoteError:   nil,
			}, nil
		}

		Expect(k8sClient.DeleteAllOf(ctx, &v1beta1.APIExtension{}, client.InNamespace(namespaceName), client.GracePeriodSeconds(5))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Secret{}, client.InNamespace(namespaceName), client.GracePeriodSeconds(5))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &fleetv1alpha1.GitRepo{}, client.InNamespace(namespaceName), client.GracePeriodSeconds(5))).To(Succeed())
		Expect(k8sClient.DeleteAllOf(ctx, &networkv1.Ingress{}, client.InNamespace(namespaceName), client.GracePeriodSeconds(5))).To(Succeed())

		// Ensure API extension is removed
		Eventually(func() bool {
			obj := &v1beta1.APIExtension{}
			err := k8sClient.Get(ctx, apiExtensionLookupKey, obj)
			// delete the target object again to make sure it is really deleted
			if err == nil {
				_ = k8sClient.Delete(ctx, obj)
			}
			return err == nil
		}, timeout, interval).Should(BeFalse())

		// Ensure Secret is removed
		Eventually(func() bool {
			obj := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretLookupKey, obj)
			// delete the target object again to make sure it is really deleted
			if err == nil {
				_ = k8sClient.Delete(ctx, obj)
			}
			return err == nil
		}, timeout, interval).Should(BeFalse())

		// Ensure GitRepo resource is removed
		Eventually(func() bool {
			obj := &fleetv1alpha1.GitRepo{}
			err := k8sClient.Get(ctx, gitrepoLookupKey, obj)
			// delete the target object again to make sure it is really deleted
			if err == nil {
				_ = k8sClient.Delete(ctx, obj)
			}
			return err == nil
		}, timeout, interval).Should(BeFalse())

		// Ensure Ingress resources are removed
		Eventually(func() bool {
			obj := &networkv1.Ingress{}
			err := k8sClient.Get(ctx, ingSvc1LookupKey, obj)
			// delete the target object again to make sure it is really deleted
			if err == nil {
				_ = k8sClient.Delete(ctx, obj)
			}
			return err == nil
		}, timeout, interval).Should(BeFalse())

		Eventually(func() bool {
			obj := &networkv1.Ingress{}
			err := k8sClient.Get(ctx, ingSvc2LookupKey, obj)
			// delete the target object again to make sure it is really deleted
			if err == nil {
				_ = k8sClient.Delete(ctx, obj)
			}
			return err == nil
		}, timeout, interval).Should(BeFalse())
	})

	When("a new APIExtension CR is created", func() {
		Context("the spec is valid", func() {
			It("should create authorization token secret if not exists", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: false,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}
				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				// Ensure token Secret resources is created
				createdSecret := &corev1.Secret{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Validate the token value matches to the one in CR
				createdAPIExtension := &v1beta1.APIExtension{}
				Expect(k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)).To(Succeed())
				Expect(createdSecret.Data["token"]).Should(Equal([]byte(createdAPIExtension.Status.TokenSecretRef.GeneratedToken)))

				// Confirm owner reference for garbage collection
				controllerRef := metav1.NewControllerRef(apiExtension, gvk)
				Expect(createdSecret.GetOwnerReferences()).Should(ContainElement(*controllerRef))

				// Ensure APIExtension TokenReady condition is True
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil && hasMatchingCondition(createdAPIExtension, metav1.Condition{
						Type:               string(v1beta1.TokenReady),
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(Clock.Now()),
						Reason:             "Test",
						Message:            "Test",
					})
				}, timeout, interval).Should(BeTrue())
			})
			It("should create repository in git if not exists", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: false,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())
				// TODO: validate generated fleet.yaml and overrides.yaml

				// Ensure APIExtension AgentRepoReady condition is True
				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil && hasMatchingCondition(createdAPIExtension, metav1.Condition{
						Type:               string(v1beta1.AgentRepoReady),
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(Clock.Now()),
						Reason:             "Test",
						Message:            "Test",
					})
				}, timeout, interval).Should(BeTrue())
			})
			It("should create Fleet GitRepo if not exists", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				// Ensure GitRepo resource is created
				createdGitRepo := &fleetv1alpha1.GitRepo{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, gitrepoLookupKey, createdGitRepo)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Get created APIExtension CR
				createdAPIExtension := &v1beta1.APIExtension{}
				Expect(k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)).To(Succeed())

				// Validate GitRepo spec is created with expected values
				Expect(createdGitRepo.Name).Should(Equal(gitrepo.Name))
				Expect(createdGitRepo.Namespace).Should(Equal(gitrepo.Namespace))
				Expect(createdGitRepo.Annotations).Should(Equal(gitrepo.Annotations))
				Expect(createdGitRepo.Spec.Repo).Should(Equal(fmt.Sprintf("%s/%s/%s.git",
					gitServer, gitUser, createdAPIExtension.UID)))
				Expect(createdGitRepo.Spec.Paths).Should(Equal(gitrepo.Spec.Paths))
				Expect(createdGitRepo.Spec.Targets).Should(Equal(gitrepo.Spec.Targets))

				// Confirm owner reference for garbage collection
				controllerRef := metav1.NewControllerRef(apiExtension, gvk)
				Expect(createdGitRepo.GetOwnerReferences()).Should(ContainElement(*controllerRef))

				// Ensure APIExtension AgentReady condition is False for AgentNotRunning
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil && hasMatchingCondition(createdAPIExtension, metav1.Condition{
						Type:               string(v1beta1.AgentReady),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.NewTime(Clock.Now()),
						Reason:             "Test",
						Message:            "Test",
					})
				}, timeout, interval).Should(BeTrue())

				// Ensure APIExtension state is Down
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						createdAPIExtension.Status.State == v1beta1.Down
				}, timeout, interval).Should(BeTrue())

				// Make GitRepo condition to Ready and verify APIExtension AgentReady condition
				// changes to True
				createdGitRepo.Status.Conditions = append(createdGitRepo.Status.Conditions,
					genericcondition.GenericCondition{
						Type:   "Ready",
						Status: corev1.ConditionTrue,
					})
				createdGitRepo.Status.ReadyClusters = 1
				Expect(k8sClient.Status().Update(ctx, createdGitRepo)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil && hasMatchingCondition(createdAPIExtension, metav1.Condition{
						Type:               string(v1beta1.AgentReady),
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(Clock.Now()),
						Reason:             "Test",
						Message:            "Test",
					})
				}, timeout, interval).Should(BeTrue())

				// Ensure APIExtension state changes to Running
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						createdAPIExtension.Status.State == v1beta1.Running
				}, timeout, interval).Should(BeTrue())
			})
			It("should contain UIExtension", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						len(createdAPIExtension.Spec.UIExtensions) == 2 &&
						createdAPIExtension.Spec.UIExtensions[0].ServiceName == svc1Name &&
						createdAPIExtension.Spec.UIExtensions[0].Label == svc1UIExtLabel &&
						createdAPIExtension.Spec.UIExtensions[0].Description == svc1UIExtDescription &&
						createdAPIExtension.Spec.UIExtensions[0].FileName == svc1UIExtFileName &&
						createdAPIExtension.Spec.UIExtensions[0].AppName == svc1UIExtAppName &&
						createdAPIExtension.Spec.UIExtensions[0].ModuleName == svc1UIExtModuleName &&
						createdAPIExtension.Spec.UIExtensions[1].ServiceName == svc2Name &&
						createdAPIExtension.Spec.UIExtensions[1].Label == svc2UIExtLabel &&
						createdAPIExtension.Spec.UIExtensions[1].Description == svc2UIExtDescription &&
						createdAPIExtension.Spec.UIExtensions[1].FileName == svc2UIExtFileName &&
						createdAPIExtension.Spec.UIExtensions[1].AppName == svc2UIExtAppName &&
						createdAPIExtension.Spec.UIExtensions[1].ModuleName == svc2UIExtModuleName
				}, timeout, interval).Should(BeTrue())
			})
			It("should contain UIExtension", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						len(createdAPIExtension.Spec.UIExtensions) == 2 &&
						createdAPIExtension.Spec.UIExtensions[0].ServiceName == svc1Name &&
						createdAPIExtension.Spec.UIExtensions[0].Label == svc1UIExtLabel &&
						createdAPIExtension.Spec.UIExtensions[0].Description == svc1UIExtDescription &&
						createdAPIExtension.Spec.UIExtensions[0].FileName == svc1UIExtFileName &&
						createdAPIExtension.Spec.UIExtensions[0].AppName == svc1UIExtAppName &&
						createdAPIExtension.Spec.UIExtensions[0].ModuleName == svc1UIExtModuleName &&
						createdAPIExtension.Spec.UIExtensions[1].ServiceName == svc2Name &&
						createdAPIExtension.Spec.UIExtensions[1].Label == svc2UIExtLabel &&
						createdAPIExtension.Spec.UIExtensions[1].Description == svc2UIExtDescription &&
						createdAPIExtension.Spec.UIExtensions[1].FileName == svc2UIExtFileName &&
						createdAPIExtension.Spec.UIExtensions[1].AppName == svc2UIExtAppName &&
						createdAPIExtension.Spec.UIExtensions[1].ModuleName == svc2UIExtModuleName
				}, timeout, interval).Should(BeTrue())
			})
			It("should create Ingress for each proxy endpoints", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				// Ensure Ingress resources are created
				createdIngress := &networkv1.Ingress{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ingSvc1LookupKey, createdIngress)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Confirm Ingress rules
				Expect(*createdIngress.Spec.IngressClassName).Should(Equal(nginxIngress))
				Expect(createdIngress.Spec.Rules).Should(Equal(ingSvc1Rules))
				Expect(createdIngress.Annotations).Should(Equal(ingSvc1Annotation))

				// Confirm owner reference for garbage collection
				controllerRef := metav1.NewControllerRef(apiExtension, gvk)
				Expect(createdIngress.GetOwnerReferences()).Should(ContainElement(*controllerRef))

				createdIngress = &networkv1.Ingress{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ingSvc2LookupKey, createdIngress)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Confirm Ingress rules
				Expect(*createdIngress.Spec.IngressClassName).Should(Equal(nginxIngress))
				Expect(createdIngress.Spec.Rules).Should(Equal(ingSvc2Rules))
				Expect(createdIngress.Annotations).Should(Equal(ingSvc2Annotation))

				// Confirm owner reference for garbage collection
				controllerRef = metav1.NewControllerRef(apiExtension, gvk)
				Expect(createdIngress.GetOwnerReferences()).Should(ContainElement(*controllerRef))

				// Ensure APIExtension IngressReady condition is True
				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil && hasMatchingCondition(createdAPIExtension, metav1.Condition{
						Type:               string(v1beta1.IngressReady),
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(Clock.Now()),
						Reason:             "Test",
						Message:            "Test",
					})
				}, timeout, interval).Should(BeTrue())
			})
			It("should become ready status eventually", func() {
				By("overridding gitclient functions so all operations succeed")
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

				// Get GitRepo resource and set condition ready
				createdGitRepo := &fleetv1alpha1.GitRepo{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, gitrepoLookupKey, createdGitRepo)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				createdGitRepo.Status.Conditions = append(createdGitRepo.Status.Conditions,
					genericcondition.GenericCondition{
						Type:   "Ready",
						Status: corev1.ConditionTrue,
					})
				createdGitRepo.Status.ReadyClusters = 1
				Expect(k8sClient.Status().Update(ctx, createdGitRepo)).To(Succeed())

				// Ensure APIExtesion status is eventually ready
				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						createdAPIExtension.Status.State == v1beta1.Running
				}, timeout, interval).Should(BeTrue())
			})
		})
		//Context("different Ingress controller is set in the configuration", func() {
		//	BeforeEach(func() {
		//		config.GetAPIExtensionConfig().IngressKind = traefikIngress
		//	})
		//	It("Should set ingress class name and rewrite annotation properly", func() {
		//		By("overridding gitclient functions so all operations succeed")
		//		gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
		//			return &mockRepository{
		//				ExistsOnRemoteValue: true,
		//				ExistsOnRemoteError: nil,
		//				InitializeError:     nil,
		//				CloneError:          nil,
		//				CommitFilesError:    nil,
		//				PushToRemoteError:   nil,
		//			}, nil
		//		}

		//		Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

		//		// Ensure Ingress resources are created
		//		createdIngress := &networkv1.Ingress{}
		//		Eventually(func() bool {
		//			err := k8sClient.Get(ctx, ingSvc1LookupKey, createdIngress)
		//			return err == nil
		//		}, timeout, interval).Should(BeTrue())

		//		// Confirm Ingress rules
		//		Expect(*createdIngress.Spec.IngressClassName).Should(Equal(traefikIngress))
		//		Expect(createdIngress.Spec.Rules).Should(Equal(ingSvc1Rules))
		//		Expect(createdIngress.Annotations).Should(Equal(ingSvc1TraefikAnnotation))

		//		// Ensure APIExtension IngressReady condition is True
		//		createdAPIExtension := &v1beta1.APIExtension{}
		//		Eventually(func() bool {
		//			err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
		//			return err == nil && hasMatchingCondition(createdAPIExtension, v1beta1.Condition{
		//				Type:   v1beta1.IngressReady,
		//				Status: metav1.ConditionTrue,
		//			})
		//		}, timeout, interval).Should(BeTrue())
		//	})
		//})
	})
	When("there is existing APIExtension CR", func() {
		var (
			createdIngress networkv1.Ingress
			createdSecret  corev1.Secret
			generatedToken string
		)

		BeforeEach(func() {
			gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
				return &mockRepository{
					ExistsOnRemoteValue: true,
					ExistsOnRemoteError: nil,
					InitializeError:     nil,
					CloneError:          nil,
					CommitFilesError:    nil,
					PushToRemoteError:   nil,
				}, nil
			}

			Expect(k8sClient.Create(ctx, apiExtension)).To(Succeed())

			// Get GitRepo resource and set condition ready
			createdGitRepo := &fleetv1alpha1.GitRepo{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, gitrepoLookupKey, createdGitRepo)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			createdGitRepo.Status.Conditions = append(createdGitRepo.Status.Conditions,
				genericcondition.GenericCondition{
					Type:   "Ready",
					Status: corev1.ConditionTrue,
				})
			createdGitRepo.Status.ReadyClusters = 1
			Expect(k8sClient.Status().Update(ctx, createdGitRepo)).To(Succeed())

			// Ensure APIExtesion status is ready
			createdAPIExtension := &v1beta1.APIExtension{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
				return err == nil &&
					createdAPIExtension.Status.State == v1beta1.Running
			}, timeout, interval).Should(BeTrue())

			generatedToken = createdAPIExtension.Status.TokenSecretRef.GeneratedToken
			Expect(k8sClient.Get(ctx, secretLookupKey, &createdSecret)).To(Succeed())
			Expect(k8sClient.Get(ctx, ingSvc1LookupKey, &createdIngress)).To(Succeed())
		})
		Context("ingress associated with the APIExtension is deleted", func() {
			It("should re-create the Ingress resource", func() {
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Delete(ctx, &createdIngress)).To(Succeed())

				// Ensure Ingress resources is recreated
				createdIngress := &networkv1.Ingress{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, ingSvc1LookupKey, createdIngress)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Verify the Ingress rules
				Expect(*createdIngress.Spec.IngressClassName).Should(Equal(nginxIngress))
				Expect(createdIngress.Spec.Rules).Should(Equal(ingSvc1Rules))
				Expect(createdIngress.Annotations).Should(Equal(ingSvc1Annotation))

				// Ensure APIExtesion status is ready
				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						createdAPIExtension.Status.State == v1beta1.Running
				}, timeout, interval).Should(BeTrue())
			})
		})
		Context("auth token Secret associated with the APIExtension is deleted", func() {
			It("should re-create the Secret with the same token value", func() {
				gitclient.NewGitClient = func(uuid string) (gitclient.Repository, error) {
					return &mockRepository{
						ExistsOnRemoteValue: true,
						ExistsOnRemoteError: nil,
						InitializeError:     nil,
						CloneError:          nil,
						CommitFilesError:    nil,
						PushToRemoteError:   nil,
					}, nil
				}

				Expect(k8sClient.Delete(ctx, &createdSecret)).To(Succeed())

				// Ensure Secret resources is recreated
				createdSecret := &corev1.Secret{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Verify the token value
				Expect(createdSecret.Data["token"]).Should(Equal([]byte(generatedToken)))

				// Ensure APIExtesion status is ready
				createdAPIExtension := &v1beta1.APIExtension{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, apiExtensionLookupKey, createdAPIExtension)
					return err == nil &&
						createdAPIExtension.Status.State == v1beta1.Running
				}, timeout, interval).Should(BeTrue())
			})
		})
	})
})

func hasMatchingCondition(a *v1beta1.APIExtension, c metav1.Condition) bool {
	if a == nil {
		return false
	}
	existingConditions := a.Status.Conditions
	for _, cond := range existingConditions {
		if c.Type == cond.Type &&
			c.Status == cond.Status {
			return true
		}
	}

	return false
}

type mockManager struct {
	endpoint       string
	serviceAccount string
	mount          string
}

func (m *mockManager) Logout(ctx context.Context, client *vault.Client) error {
	return nil
}

func (m *mockManager) GetVaultClient(ctx context.Context) (*vault.Client, error) {
	return nil, nil
}

func (m *mockManager) GetKVSecret(ctx context.Context, client *vault.Client, path string) (*vault.KVSecret, error) {
	return nil, nil
}

func (m *mockManager) GetSecretValueString(ctx context.Context, client *vault.Client, path string, key string) (string, error) {
	switch path {
	case utils.GetSecretServiceHarborServicePath():
		switch key {
		case utils.GetSecretServiceHarborServiceKVKeyCert():
			return generateTestCACertificate()
		case utils.GetSecretServiceHarborServiceKVKeyUsername():
			return "test-username", nil
		case utils.GetSecretServiceHarborServiceKVKeyPassword():
			return "test-password", nil
		}
	}

	return "", fmt.Errorf("mock err: path %s, key %s", path, key)
}

func newMockManager(endpoint, serviceAccount, mount string) manager.Manager {
	return &mockManager{
		endpoint:       endpoint,
		serviceAccount: serviceAccount,
		mount:          mount,
	}
}

func generateTestCACertificate() (string, error) {
	// Generate a new private key
	caPrivateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return "", err
	}

	// Create a self-signed CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"My CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertificateDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return "", err
	}
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertificateDER})
	caCertBase64 := base64.StdEncoding.EncodeToString(caCertPEM)

	return caCertBase64, nil
}
