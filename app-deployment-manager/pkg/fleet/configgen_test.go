// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	yamlv3 "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kustomize "sigs.k8s.io/kustomize/api/types"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var namespacesTest = []v1beta1.Namespace{
	{
		Name:        "ns-test",
		Labels:      map[string]string{"test-label-key": "test-label-value"},
		Annotations: map[string]string{"test-ann-key": "test-ann-value"},
	},
	{
		Name:        "ns-test1",
		Labels:      map[string]string{"test-label-key1": "test-label-value1"},
		Annotations: map[string]string{"test-ann-key1": "test-ann-value1"},
	},
}

// Verify generated fleet.yaml file contents
func verifyFleetBasicConfig(filename string, app v1beta1.Application, d *v1beta1.Deployment) {
	fleetConfigStruct := &Config{}
	fleetconf, err := os.ReadFile(filename)
	Expect(err).To(BeNil())
	Expect(yamlv3.Unmarshal(fleetconf, fleetConfigStruct)).To(Succeed())

	repo := app.HelmApp.Repo
	u, err := url.Parse(repo)
	Expect(err).To(BeNil())

	chartUrl, _ := url.JoinPath(app.HelmApp.Repo, app.HelmApp.Chart)
	bundlename := BundleName(app, d.GetName())
	Expect(string(fleetConfigStruct.Name)).To(Equal(bundlename))
	Expect(string(fleetConfigStruct.Helm.ReleaseName)).To(Equal(bundlename))

	repoUrl := fleetConfigStruct.Helm.Repo
	if u.Scheme == "oci" {
		Expect(repoUrl).To(Equal(""))
		Expect(string(fleetConfigStruct.Helm.Chart)).To(Equal(chartUrl))
	} else {
		Expect(repoUrl).To(Equal(app.HelmApp.Repo))
		Expect(string(fleetConfigStruct.Helm.Chart)).To(Equal(app.HelmApp.Chart))
	}

	Expect(string(fleetConfigStruct.Helm.Version)).To(Equal(app.HelmApp.Version))
}

// Verify generated namespace fleet.yaml file contents
func verifyFleetNamespaceConfig(filename string, ns v1beta1.Namespace) {
	fleetConfigStruct := &Config{}
	fleetconf, err := os.ReadFile(filename)
	Expect(err).To(BeNil())
	Expect(yamlv3.Unmarshal(fleetconf, fleetConfigStruct)).To(Succeed())

	// Verify the added details
	Expect(string(fleetConfigStruct.Name)).To(ContainSubstring(ns.Name))
	Expect(string(fleetConfigStruct.DefaultNamespace)).To(Equal(ns.Name))

	for k := range fleetConfigStruct.NamespaceLabels {
		Expect(string(fleetConfigStruct.NamespaceLabels[k])).To(Equal(ns.Labels[k]))
	}

	for k := range fleetConfigStruct.NamespaceAnnotations {
		Expect(string(fleetConfigStruct.NamespaceAnnotations[k])).To(Equal(ns.Annotations[k]))
	}

	// Expect helm section to be blank
	Expect(fleetConfigStruct.Helm.ReleaseName).To(Equal(""))
	Expect(fleetConfigStruct.Helm.Repo).To(Equal(""))
	Expect(fleetConfigStruct.Helm.Chart).To(Equal(""))
	Expect(fleetConfigStruct.Helm.Version).To(Equal(""))
}

var _ = Describe("Fleet config generator", func() {
	var (
		deployment *v1beta1.Deployment
		secrets    []corev1.Secret
		profValues []byte
		overValues []byte
		globValues *ExtraValues

		imageRegistry = "oci://test.registry.intel.com/catalog-apps-mock-org-1-mock-project-1-in-mock-org-1"
		imageRegUser  = "user"
		imageRegPass  = "pass"

		kustomizationLiterals []string
		fleetConfigStruct     *Config
	)

	BeforeEach(func() {
		// Ensure secrets for profile, overriding values and image registry
		// credentials available before running the tests
		profValues = []byte(`{"prof-key1":"prof-value1","prof-key2":"prof-value2"}`)
		overValues = []byte(`{"over-key1":"over-value1","over-key2":"over-value2"}`)

		secrets = []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profilesecret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": profValues,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profilesecretwithcred",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": []byte(`{"imagePullPolicy": "%GeneratedDockerCredential%"}`),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profilesecretwithprehook",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": []byte(`{"imagePullPolicy": "%GeneratedDockerCredential%","preHook": "%PreHookCredential%"}`),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valuessecret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": overValues,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valuessecretwithimagepullpolicy",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": []byte(`{"imagePullPolicy": "%GeneratedDockerCredential%"}`),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "imageregcreds",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte(imageRegUser),
					"password": []byte(imageRegPass),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profsecretwithimageregistry",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"values": []byte(`{"image": "%ImageRegistryURL%/some-image", "current-org": "%OrgName%","current-project": "%ProjectName%"}`),
				},
			},
		}

		for i := range secrets {
			sec := secrets[i]
			key := types.NamespacedName{Name: sec.Name, Namespace: sec.Namespace}
			err := k8sClient.Get(ctx, key, &corev1.Secret{})
			if apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, &sec)).To(Succeed())
			}
		}

		deployment = &v1beta1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "wordpress-deployment",
			Namespace: "default",
			UID:       "447107d5-a622-404b-b7b1-260131c1b5a9",
			Labels: map[string]string{
				string(v1beta1.AppOrchActiveProjectID): "64f42b12-af68-4676-a689-657dd670daab",
				"app.kubernetes.io/created-by":         "app-deployment-manager",
				"app.kubernetes.io/instance":           "wordpress-deployment",
				"app.kubernetes.io/managed-by":         "kustomize",
				"app.kubernetes.io/name":               "deployment",
				"app.kubernetes.io/part-of":            "app-deployment-manager",
			},
			Generation: 1,
		},
			Spec: v1beta1.DeploymentSpec{
				DisplayName:    "My Wordpress Blog",
				Project:        "test-project",
				DeploymentType: "auto-scaling",
				DeploymentPackageRef: v1beta1.DeploymentPackageRef{
					Namespaces: []v1beta1.Namespace{},
				},
				Applications: []v1beta1.Application{
					{
						Name:      "wordpress",
						Version:   "0.1.0",
						Namespace: "apps",
						Targets: []map[string]string{{
							"color": "blue",
						}},
						HelmApp: &v1beta1.HelmApp{
							Chart:   "wordpress",
							Version: "15.2.42",
							Repo:    "https://charts.bitnami.com/bitnami",
						},
					},
					{
						Name:      "dependency1",
						Version:   "0.1.0",
						Namespace: "apps",
						Targets: []map[string]string{{
							"color": "blue",
						}},
						HelmApp: &v1beta1.HelmApp{
							Chart:   "dependency",
							Version: "0.1.0",
							Repo:    "https://charts.bitnami.com/bitnami",
						},
					},
				},
			},
		}

		imageRegistryURL, err := url.Parse(imageRegistry)
		Expect(err).To(BeNil())
		credjson, err := json.Marshal(map[string]interface{}{
			"auths": map[string]interface{}{
				imageRegistryURL.Host: map[string]string{
					"username": imageRegUser,
					"password": imageRegPass,
				},
			},
		})
		Expect(err).To(BeNil())
		kustomizationLiterals = []string{
			fmt.Sprintf(`.dockerconfigjson=%s`, credjson),
			fmt.Sprintf("accessKeyId=%s", imageRegUser),
			fmt.Sprintf("secretKey=%s", imageRegPass),
		}

		globValues = &ExtraValues{
			Global: GlobalValues{
				Fleet: GlobalValuesFleet{
					DeploymentGeneration: 1,
					ClusterLabels: map[string]string{
						string(v1beta1.ClusterName):    fmt.Sprintf("global.fleet.clusterLabels.%s", v1beta1.ClusterName),
						string(v1beta1.FleetProjectID): fmt.Sprintf("global.fleet.clusterLabels.%s", v1beta1.FleetProjectID),
						string(v1beta1.FleetHostUuid):  fmt.Sprintf("global.fleet.%s", v1beta1.FleetHostUuid),
					},
				},
			},
		}

		os.Setenv("FLEET_ADD_GLOBAL_VARS", "true")
	})

	When("generating fleet configs for a deployment spec", func() {
		Context("only required fields are in the spec", func() {
			It("should create fleet.yaml, fleet-globals.yaml and empty profiles.yaml and overrides.yaml", func() {
				// Generate Fleet Configs
				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				app := deployment.Spec.Applications[0]

				// Validate fleet.yaml
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, app, deployment)

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate overrides.yaml is empty
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))
			})
		})

		Context("multiple applications in the spec", func() {
			It("should create fleet.yaml, fleet-globals.yaml and empty profiles.yaml and overrides.yaml for each application", func() {
				// Generate Fleet Configs
				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				for _, app := range deployment.Spec.Applications {
					// Validate fleet.yaml
					fleetdir := filepath.Join(basedir, app.Name)
					yaml := filepath.Join(fleetdir, "fleet.yaml")
					Expect(yaml).Should(BeAnExistingFile())
					verifyFleetBasicConfig(yaml, app, deployment)

					// Validate profile.yaml is empty
					yaml = filepath.Join(fleetdir, "profile.yaml")
					Expect(yaml).Should(BeAnExistingFile())
					contents, err := os.ReadFile(yaml)
					Expect(err).To(BeNil())
					Expect(string(contents)).To(Equal(""))

					// Validate overrides.yaml is empty
					yaml = filepath.Join(fleetdir, "overrides.yaml")
					Expect(yaml).Should(BeAnExistingFile())
					contents, err = os.ReadFile(yaml)
					Expect(err).To(BeNil())
					Expect(string(contents)).To(Equal(""))

					// Validate fleet-globals.yaml exists
					yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
					Expect(yaml).Should(BeAnExistingFile())
					contents, err = os.ReadFile(yaml)
					Expect(err).To(BeNil())
					contentInExtraValues := &ExtraValues{}
					Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
					Expect(contentInExtraValues).To(Equal(globValues))
				}
			})
		})

		Context("ProfileSecretName is provided", func() {
			It("should create profiles.yaml with the secret contents", func() {
				app := &deployment.Spec.Applications[0]
				app.ProfileSecretName = "profilesecret"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Validate profile.yaml contents
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(string(profValues)))

				// Validate overrides.yaml is empty
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))
			})
		})

		Context("ProfileSecretName is provided when secret does not exists", func() {
			It("should return error", func() {
				app := &deployment.Spec.Applications[0]
				app.ProfileSecretName = "wrongprofilesecret"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).NotTo(Succeed())
			})
		})

		Context("ValuesSecretName is provided", func() {
			It("should create overrides.yaml with the secret contents", func() {
				app := &deployment.Spec.Applications[0]
				app.ValueSecretName = "valuessecret"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate overrides.yaml contents
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(string(overValues)))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))
			})
		})

		Context("ValuesSecretName is provided when secret does not exists", func() {
			It("should return error", func() {
				app := &deployment.Spec.Applications[0]
				app.ValueSecretName = "wrongvaluessecret"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).NotTo(Succeed())
			})
		})

		Context("PreHook image credential is provided", func() {
			It("should create new fleet.yaml with image registry creds", func() {
				app := &deployment.Spec.Applications[0]
				app.HelmApp.ImageRegistry = imageRegistry
				app.HelmApp.ImageRegistrySecretName = "imageregcreds"
				app.ProfileSecretName = "profilesecretwithprehook"
				bundleName := BundleName(*app, deployment.GetName())

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Validate fleet.yaml has additional kustomization.dir configuration
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, &fleetConfigStruct)).To(Succeed())
				Expect(string(fleetConfigStruct.Kustomize.Dir)).To(Equal("./kustomize"))

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				values := fmt.Sprintf(`{"imagePullPolicy": "%s","preHook": "%s"}`, bundleName, "%PreHookCredential%")
				Expect(string(contents)).To(Equal(values))

				// Validate overrides.yaml has bundle name as image pull policy secret name
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))

				// Validate kustomization.yaml contents
				yaml = filepath.Join(fleetdir, "kustomize", "kustomization.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				_, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())

				// Validate secret-dir contents
				yaml = filepath.Join(fleetdir, "secret-dir", "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				_, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())

				// Validate secret-dir contents
				yaml = filepath.Join(fleetdir, "secret-dir", "image-reg-secret.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				secretVal := &SecretSpec{}
				Expect(yamlv3.Unmarshal(contents, secretVal)).To(Succeed())
				Expect(secretVal.Metadata.Name).To(Equal(bundleName))
				Expect(secretVal.Type).To(Equal("kubernetes.io/dockerconfigjson"))

				km := &kustomize.Kustomization{}
				Expect(yamlv3.Unmarshal(contents, &km)).To(Succeed())
			})
		})

		Context("Namespaces are provided in the deployment package", func() {
			It("should create 1 separate dir and 1 new fleet.yaml with namespace details only", func() {
				app := &deployment.Spec.Applications[0]

				deployment.Spec.DeploymentPackageRef.Namespaces = make([]v1beta1.Namespace, 1)
				deployment.Spec.DeploymentPackageRef.Namespaces[0] = namespacesTest[0]

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Expect that fleet.yaml has namespace's fleet.yaml as dependsOn
				fleetConfigStruct := &Config{}
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, fleetConfigStruct)).To(Succeed())
				Expect(fleetConfigStruct.DependsOn[0].Name).To(ContainSubstring(namespacesTest[0].Name))

				// Validate namespace's fleet.yaml basic contents
				fleetdir = filepath.Join(basedir, app.Name)
				yaml = filepath.Join(fleetdir, namespacesTest[0].Name+"-ns/fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetNamespaceConfig(yaml, namespacesTest[0])
			})

			It("should create 2 separate dirs and 2 new fleet.yaml with namespace details only", func() {
				app := &deployment.Spec.Applications[0]

				deployment.Spec.DeploymentPackageRef.Namespaces = make([]v1beta1.Namespace, 2)
				deployment.Spec.DeploymentPackageRef.Namespaces[0] = namespacesTest[0]
				deployment.Spec.DeploymentPackageRef.Namespaces[1] = namespacesTest[1]

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Expect that fleet.yaml has namespace's fleet.yaml as dependsOn
				fleetConfigStruct := &Config{}
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, fleetConfigStruct)).To(Succeed())
				Expect(fleetConfigStruct.DependsOn[0].Name).To(ContainSubstring(deployment.Spec.DeploymentPackageRef.Namespaces[0].Name))
				Expect(fleetConfigStruct.DependsOn[1].Name).To(ContainSubstring(deployment.Spec.DeploymentPackageRef.Namespaces[1].Name))

				// Validate 1st namespace's fleet.yaml basic contents
				fleetdir = filepath.Join(basedir, app.Name)
				yaml = filepath.Join(fleetdir, deployment.Spec.DeploymentPackageRef.Namespaces[0].Name+"-ns/fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetNamespaceConfig(yaml, deployment.Spec.DeploymentPackageRef.Namespaces[0])

				// Validate 2nd namespace's fleet.yaml basic contents
				fleetdir = filepath.Join(basedir, app.Name)
				yaml = filepath.Join(fleetdir, deployment.Spec.DeploymentPackageRef.Namespaces[1].Name+"-ns/fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetNamespaceConfig(yaml, deployment.Spec.DeploymentPackageRef.Namespaces[1])
			})
		})

		Context("ImageRegistrySecretName is provided", func() {
			It("should create kustomization config with the image registry creds", func() {
				app := &deployment.Spec.Applications[0]
				app.HelmApp.ImageRegistry = imageRegistry
				app.HelmApp.ImageRegistrySecretName = "imageregcreds"
				app.ValueSecretName = "valuessecretwithimagepullpolicy"
				bundleName := BundleName(*app, deployment.GetName())

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Validate fleet.yaml has additional kustomization.dir configuration
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, &fleetConfigStruct)).To(Succeed())
				Expect(string(fleetConfigStruct.Kustomize.Dir)).To(Equal("./kustomize"))

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate overrides.yaml has bundle name as image pull policy secret name
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				values := fmt.Sprintf(`{"imagePullPolicy": "%s"}`, bundleName)
				Expect(string(contents)).To(Equal(values))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))

				// Validate kustomization.yaml contents
				yaml = filepath.Join(fleetdir, "kustomize", "kustomization.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())

				km := &kustomize.Kustomization{}
				Expect(yamlv3.Unmarshal(contents, &km)).To(Succeed())

				Expect(km.SecretGenerator[0].Namespace).To(Equal("apps"))
				Expect(km.SecretGenerator[0].Name).To(Equal(bundleName))
				Expect(km.SecretGenerator[0].Type).To(Equal("kubernetes.io/dockerconfigjson"))
				Expect(km.SecretGenerator[0].LiteralSources).Should(ConsistOf(kustomizationLiterals))

			})
		})

		Context("ImageRegistrySecretName is empty but GeneratedDockerCredential is added in values file", func() {
			It("should return error", func() {
				app := &deployment.Spec.Applications[0]
				app.ProfileSecretName = "profilesecretwithcred"
				app.HelmApp.ImageRegistry = ""
				app.HelmApp.ImageRegistrySecretName = ""

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).NotTo(Succeed())
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).ShouldNot(BeAnExistingFile())
			})
		})

		Context("ImageRegistrySecretName is provided when secret does not exists", func() {
			It("should return error", func() {
				app := &deployment.Spec.Applications[0]
				app.HelmApp.ImageRegistry = imageRegistry
				app.HelmApp.ImageRegistrySecretName = "wrongimageregcreds"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).NotTo(Succeed())
			})
		})

		Context("ImageRegistryURL% is provided", func() {
			It("should replace them in values", func() {
				app := &deployment.Spec.Applications[0]
				app.HelmApp.ImageRegistry = imageRegistry
				app.HelmApp.ImageRegistrySecretName = "imageregcreds"
				app.ProfileSecretName = "profsecretwithimageregistry"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate profile.yaml is empty
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())

				Expect(string(contents)).To(
					Equal(
						`{"image": "test.registry.intel.com/catalog-apps-mock-org-1-mock-project-1-in-mock-org-1/some-image", "current-org": "mock-org-1","current-project": "mock-project-1-in-mock-org-1"}`))
			})
		})

		Context("ImageRegistry tag is provided by no image registry object present", func() {
			It("should throw error", func() {
				app := &deployment.Spec.Applications[0]
				app.HelmApp.ImageRegistrySecretName = "imageregcreds"
				app.ProfileSecretName = "profsecretwithimageregistry"

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).
					To(MatchError("imageRegistry not set but '%ImageRegistryURL%' tag is present"))

			})
		})

		Context("DependsOn is provided", func() {
			It("should create fleet.yaml with dependsOn applications", func() {
				// Set dependency1 as dependency of word-press application
				app0 := &deployment.Spec.Applications[0]
				app1 := &deployment.Spec.Applications[1]
				app0.DependsOn = []string{"dependency1"}

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app0.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app0, deployment)

				// Validate fleet.yaml has additional dependsOn configuration
				dependsOnValue := BundleName(*app1, deployment.GetName())
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, &fleetConfigStruct)).To(Succeed())
				Expect(string(fleetConfigStruct.DependsOn[0].Name)).To(Equal(dependsOnValue))

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate overrides.yaml is empty
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))
			})
		})

		Context("IgnoredResources is provided", func() {
			It("should create fleet.yaml with diff configuration", func() {
				app := &deployment.Spec.Applications[0]
				app.IgnoreResources = []v1beta1.IgnoreResource{
					{
						Name: "cfg",
						Kind: "ConfigMap",
					},
					{
						Name:      "cfg2",
						Kind:      "ConfigMap",
						Namespace: "test-ns",
					},
					{
						Name: "ValWebConf",
						Kind: "ValidatingWebhookConfiguration",
					},
					{
						Name: "MutWebConf",
						Kind: "MutatingWebhookConfiguration",
					},
					{
						Name: "secretVal",
						Kind: "Secret",
					},
					{
						Name:      "secretVal2",
						Kind:      "Secret",
						Namespace: "test-ns",
					},
					{
						Name: "crdVal",
						Kind: "CustomResourceDefinition",
					},
					{
						Name:      "envFilter",
						Kind:      "EnvoyFilter",
						Namespace: "istio-ns",
					},
				}

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).To(Succeed())

				// Validate fleet.yaml basic contents
				fleetdir := filepath.Join(basedir, app.Name)
				yaml := filepath.Join(fleetdir, "fleet.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				verifyFleetBasicConfig(yaml, *app, deployment)

				// Validate fleet.yaml has additional diff configuration
				fleetconf, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(yamlv3.Unmarshal(fleetconf, &fleetConfigStruct)).To(Succeed())
				Expect(&fleetConfigStruct.Diff.ComparePatches).NotTo(BeNil())

				comparePatches := fleetConfigStruct.Diff.ComparePatches
				Expect(comparePatches).To(HaveLen(len(app.IgnoreResources)))

				for _, ignoreResource := range app.IgnoreResources {
					for _, patch := range comparePatches {
						if patch.Name == ignoreResource.Name && patch.Kind == ignoreResource.Kind &&
							(patch.Namespace == "" || patch.Namespace == ignoreResource.Namespace) {
							switch patch.Kind {
							case "ConfigMap", "Secret":
								Expect(patch.APIVersion).To(Equal("v1"))
							case "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration":
								Expect(patch.APIVersion).To(Equal("admissionregistration.k8s.io/v1"))
							}
							continue
						}
					}
				}

				// Validate profile.yaml is empty
				yaml = filepath.Join(fleetdir, "profile.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err := os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate overrides.yaml is empty
				yaml = filepath.Join(fleetdir, "overrides.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal(""))

				// Validate fleet-globals.yaml exists
				yaml = filepath.Join(fleetdir, "fleet-globals.yaml")
				Expect(yaml).Should(BeAnExistingFile())
				contents, err = os.ReadFile(yaml)
				Expect(err).To(BeNil())
				contentInExtraValues := &ExtraValues{}
				Expect(yamlv3.Unmarshal(contents, contentInExtraValues)).To(Succeed())
				Expect(contentInExtraValues).To(Equal(globValues))
			})
		})

		Context("IgnoredResources has invalid Kind", func() {
			It("should throw error when creating diff configuration", func() {
				app := &deployment.Spec.Applications[0]
				app.IgnoreResources = []v1beta1.IgnoreResource{
					{
						Name: "testResource",
						Kind: "UnsupportedKind",
					},
				}

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).
					To(MatchError("unsupported: Kind UnsupportedKind not supported in diff configuration"))
			})
		})

		Context("IgnoredResources has unnecessary namespace", func() {
			It("should throw error when creating diff configuration", func() {
				app := &deployment.Spec.Applications[0]
				app.IgnoreResources = []v1beta1.IgnoreResource{
					{
						Name: "testWebHook",
						Kind: "ValidatingWebhookConfiguration",
						Namespace: "test-ns", // ValidatingWebhookConfiguration should not have namespace
					},
				}

				basedir := "/tmp/fleet0"
				Expect(os.RemoveAll(basedir)).To(Succeed())
				Expect(GenerateFleetConfigs(deployment, basedir, k8sClient, nexusClient.RuntimeprojectEdgeV1())).
					To(MatchError("namespace is not supported for ValidatingWebhookConfiguration kind in diff configuration"))
			})
		})
	})
})
