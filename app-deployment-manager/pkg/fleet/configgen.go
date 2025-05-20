// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"context"
	"crypto/sha1" // #nosec G505 todo: use different sha algorithm
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned/typed/runtimeproject.edge-orchestrator.intel.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage/names"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kustomize "sigs.k8s.io/kustomize/api/types"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
)

type BundleType int

const (
	// This special string is a placeholder for the actual credential name in
	// the Helm profile.yaml and overrides.yaml
	CredentialString    string = "%GeneratedDockerCredential%"
	PreHookString       string = "%PreHookCredential%"
	ImageRegistryURL    string = "%ImageRegistryURL%"
	RegistryProjectName        = "%RegistryProjectName%"

	BundleTypeUnknown BundleType = 0
	BundleTypeInit    BundleType = 1
	BundleTypeApp     BundleType = 2

	BundleTypeUnknownString = "unknown"
	BundleTypeInitString    = "init"
	BundleTypeAppString     = "app"

	NexusOrgLabel         = "runtimeorgs.runtimeorg.edge-orchestrator.intel.com"
	RegistryProjectPrefix = "catalog-apps" // Corresponds to HarborProjectName in TenantController
)

var (
	IgnoreConfigMapOp = []Operation{
		{
			Op:   "remove",
			Path: "/metadata/annotations",
		},
		{
			Op:   "remove",
			Path: "/data",
		},
	}
	IgnoreVWCOp = []Operation{
		{
			Op:   "remove",
			Path: "/webhooks",
		},
	}
	IgnoreMWCOp = []Operation{
		{
			Op:   "remove",
			Path: "/webhooks",
		},
	}
	IgnoreSecretOp = []Operation{
		{
			Op:   "remove",
			Path: "/data",
		},
		{
			Op:   "remove",
			Path: "/metadata",
		},
	}
	IgnoreCRDOp = []Operation{
		{
			Op:   "remove",
			Path: "/spec",
		},
	}
	IgnoreEnvoyOp = []Operation{
		{
			Op:   "remove",
			Path: "/spec/configPatches",
		},
	}
	IgnoreDeploymentOp = []Operation{
		{
			Op:   "remove",
			Path: "/spec/template/spec",
		},
	}
	IgnoreJobOp = []Operation{
		{
			Op:   "remove",
			Path: "/spec/template/spec",
		},
	}
)

func (b BundleType) String() string {
	return [...]string{BundleTypeUnknownString, BundleTypeInitString, BundleTypeAppString}[b]
}

type DiffOptions struct {
	ComparePatches []ComparePatch `yaml:"comparePatches,omitempty"`
}

type Operation struct {
	// Op is usually "remove"
	// +nullable
	Op string `yaml:"op,omitempty"`
	// Path is the JSON path to remove.
	// +nullable
	Path string `yaml:"path,omitempty"`
	// Value is usually empty.
	// +nullable
	Value string `yaml:"value,omitempty"`
}

type ComparePatch struct {
	// Kind is the kind of the resource to match.
	// +nullable
	Kind string `yaml:"kind,omitempty"`
	// APIVersion is the apiVersion of the resource to match.
	// +nullable
	APIVersion string `yaml:"apiVersion,omitempty"`
	// Namespace is the namespace of the resource to match.
	// +nullable
	Namespace string `yaml:"namespace,omitempty"`
	// Name is the name of the resource to match.
	// +nullable
	Name string `yaml:"name,omitempty"`
	// Operations remove a JSON path from the resource.
	// +nullable
	Operations []Operation `yaml:"operations,omitempty"`
	// JSONPointers ignore diffs at a certain JSON path.
	// +nullable
	JSONPointers []string `yaml:"jsonPointers,omitempty"`
}

type Config struct {
	Name               string
	Labels             DeployLabels
	DefaultNamespace   string `yaml:"defaultNamespace"`
	DeleteCRDResources bool   `yaml:"deleteCRDResources"`
	Helm               HelmApp
	Kustomize          struct {
		Dir string
	} `yaml:"kustomize,omitempty"`
	DependsOn            []DependsOnItem   `yaml:"dependsOn"`
	Diff                 *DiffOptions      `yaml:"diff,omitempty"`
	NamespaceLabels      map[string]string `yaml:"namespaceLabels,omitempty"`
	NamespaceAnnotations map[string]string `yaml:"namespaceAnnotations,omitempty"`
}

type DeployLabels struct {
	DeploymentID         string `yaml:"app.edge-orchestrator.intel.com/deployment-id,omitempty"`
	DeploymentGeneration string `yaml:"deploymentGeneration,omitempty"`
	AppName              string `yaml:"app.edge-orchestrator.intel.com/app-name,omitempty"`
	BundleType           string `yaml:"app.edge-orchestrator.intel.com/bundle-type,omitempty"`
}

type PodSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}

type Metadata struct {
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}
type PolicyRule struct {
	From []struct {
		PodSelector PodSelector `yaml:"podSelector,omitempty"`
	} `yaml:"from,omitempty"`
	To []struct {
		PodSelector PodSelector `yaml:"podSelector,omitempty"`
	} `yaml:"to,omitempty"`
}

type PolicySpec struct {
	PodSelector PodSelector  `yaml:"podSelector,omitempty"`
	PolicyTypes []string     `yaml:"policyTypes,omitempty"`
	Ingress     []PolicyRule `yaml:"ingress,omitempty"`
	Egress      []PolicyRule `yaml:"egress,omitempty"`
}

type SecretSpec struct {
	APIVersion string            `yaml:"apiVersion,omitempty"`
	Kind       string            `yaml:"kind,omitempty"`
	Type       string            `yaml:"type,omitempty"`
	Metadata   Metadata          `yaml:"metadata,omitempty"`
	Data       map[string][]byte `yaml:"data,omitempty"`
}
type NetworkPolicy struct {
	APIVersion string     `yaml:"apiVersion,omitempty"`
	Kind       string     `yaml:"kind,omitempty"`
	Metadata   Metadata   `yaml:"metadata,omitempty"`
	Spec       PolicySpec `yaml:"spec,omitempty"`
}

type HelmApp struct {
	ReleaseName string `yaml:"releaseName"`
	Repo        string
	Chart       string
	Version     string
	ValuesFiles []string `yaml:"valuesFiles"`
}

type DependsOnItem struct {
	Name     string   `yaml:"name,omitempty"`
	Selector Selector `yaml:"selector,omitempty"`
}

type Selector struct {
	MatchLabels DeployLabels `yaml:"matchLabels,omitempty"`
}

type ExtraValues struct {
	Global GlobalValues
}

type GlobalValues struct {
	Fleet GlobalValuesFleet
}

type GlobalValuesFleet struct {
	DeploymentGeneration int64             `yaml:"deploymentGeneration"`
	ClusterLabels        map[string]string `yaml:"clusterLabels"`
}

// GenerateFleetConfigs generates fleet configurations for the applications to a given path
func GenerateFleetConfigs(d *v1beta1.Deployment, baseDir string, kc client.Client, nc nexus.RuntimeProjectsGetter) error {
	appMap := map[string]v1beta1.Application{}
	var initNsBundleName string

	for _, app := range d.Spec.Applications {
		appMap[app.Name] = app
	}

	// Only generate name if dp.Namespaces defined
	if len(d.Spec.DeploymentPackageRef.Namespaces) > 0 {
		initNsBundleName = names.SimpleNameGenerator.GenerateName("")
	}

	registryProjectName, err := getRegistryProjectName(d, nc)
	if err != nil {
		return err
	}

	for _, app := range d.Spec.Applications {
		// Get default namespace
		namespace := app.Namespace

		hasImageCreds := app.HelmApp.ImageRegistrySecretName != ""
		hasPreHook := false
		bundleName := BundleName(app, d.GetName())

		// Create Config
		fleetConf := newFleetConfig(app.Name, appMap, d.GetId(), d.GetName(), d.GetGeneration(), namespace)
		fleetPath := filepath.Join(baseDir, app.Name)

		// Generate profile.yaml with profile contents
		profileyaml := "profile.yaml"
		contents, err := valuesFromSecret(app.ProfileSecretName, d.Namespace, kc)
		if err != nil {
			return err
		}
		if hasImageCreds {
			contents = strings.Replace(contents, CredentialString, bundleName, -1)
		}

		if strings.Contains(contents, CredentialString) {
			err := errors.New("token string present without Docker credentials")
			return err
		}

		// Check if Prehook present and if it needs image pull.
		if strings.Contains(contents, PreHookString) {
			hasPreHook = true
		}

		if strings.Contains(contents, ImageRegistryURL) {
			if app.HelmApp.ImageRegistry == "" {
				return fmt.Errorf("imageRegistry not set but '%s' tag is present", ImageRegistryURL)
			}
			imageRegistryURLBare := strings.TrimPrefix(app.HelmApp.ImageRegistry, "oci://")
			imageRegistryURLBare = strings.TrimPrefix(imageRegistryURLBare, "http://")
			imageRegistryURLBare = strings.TrimPrefix(imageRegistryURLBare, "https://")
			imageRegistryURLBare = strings.TrimSuffix(imageRegistryURLBare, "/")
			log.Debugf("Replacing: %s in App %s with %s", ImageRegistryURL, app.Name, imageRegistryURLBare)
			contents = strings.Replace(contents, ImageRegistryURL, imageRegistryURLBare, -1)
		}

		if strings.Contains(contents, RegistryProjectName) {
			log.Debugf("Replacing: %s in App %s with %s", RegistryProjectName, app.Name, registryProjectName)

			contents = strings.Replace(contents, RegistryProjectName, registryProjectName, -1)
		}

		err = utils.WriteFile(fleetPath, profileyaml, []byte(contents))
		if err != nil {
			return err
		}
		fleetConf.Helm.ValuesFiles = append(fleetConf.Helm.ValuesFiles, profileyaml)

		// Generate overrides.yaml with deployment time overriding values
		overridesyaml := "overrides.yaml"
		contents, err = valuesFromSecret(app.ValueSecretName, d.Namespace, kc)
		if err != nil {
			return err
		}
		if hasImageCreds {
			contents = strings.Replace(contents, CredentialString, bundleName, -1)
		}
		err = utils.WriteFile(fleetPath, overridesyaml, []byte(contents))
		if err != nil {
			return err
		}
		fleetConf.Helm.ValuesFiles = append(fleetConf.Helm.ValuesFiles, overridesyaml)

		// Generate fleet-globals.yaml. ADM Deployment controller relies on
		// global.fleet.clusterLabels values being present in the
		// BundleDeployment. Fleet v0.5.0 added them automatically but v0.6.0
		// does not. We explicitly add them below for 0.6.0.
		globalyaml := "fleet-globals.yaml"
		extraValue := ExtraValues{
			Global: GlobalValues{
				Fleet: GlobalValuesFleet{
					DeploymentGeneration: d.GetGeneration(),
				},
			},
		}
		if g, ok := os.LookupEnv("FLEET_ADD_GLOBAL_VARS"); ok && g == "true" {
			extraValue.Global.Fleet.ClusterLabels = map[string]string{
				string(v1beta1.ClusterName): fmt.Sprintf("global.fleet.clusterLabels.%s",
					v1beta1.ClusterName),
				string(v1beta1.FleetProjectID): fmt.Sprintf("global.fleet.clusterLabels.%s",
					v1beta1.FleetProjectID),
				string(v1beta1.FleetHostUuid): fmt.Sprintf("global.fleet.%s",
					v1beta1.FleetHostUuid),
			}
		}
		err = WriteExtraValues(fleetPath, globalyaml, &extraValue)
		if err != nil {
			return err
		}

		fleetConf.Helm.ValuesFiles = append(fleetConf.Helm.ValuesFiles, globalyaml)

		ingressName := app.Name + "-" + app.Version + "-" + strings.ReplaceAll(d.Name, "deployment-", "") + "-ingress" // max length: 61
		egressName := app.Name + "-" + app.Version + "-" + strings.ReplaceAll(d.Name, "deployment-", "") + "-egress"   // max length: 60
		policy := PolicyRule{}
		ingressPolicy := NetworkPolicy{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
			Metadata: Metadata{
				Name: ingressName,
			},
			Spec: PolicySpec{
				PodSelector: PodSelector{},
				Ingress:     []PolicyRule{policy},
				PolicyTypes: []string{"Ingress"},
			},
		}

		// Generate fleet.yaml from Config
		err = WriteNetworkPolicyConfig(
			filepath.Join(fleetPath, "kustomize"),
			ingressPolicy, "network-policy-ingress.yaml")
		if err != nil {
			return err
		}

		egressPolicy := NetworkPolicy{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
			Metadata: Metadata{
				Name: egressName,
			},
			Spec: PolicySpec{
				PodSelector: PodSelector{},
				Egress:      []PolicyRule{policy},
				PolicyTypes: []string{"Egress"},
			},
		}

		// Generate fleet.yaml from Config
		err = WriteNetworkPolicyConfig(
			filepath.Join(fleetPath, "kustomize"),
			egressPolicy, "network-policy-egress.yaml")
		if err != nil {
			return err
		}

		k := &kustomize.Kustomization{
			TypeMeta: kustomize.TypeMeta{
				Kind:       kustomize.KustomizationKind,
				APIVersion: kustomize.KustomizationVersion,
			},
		}

		k.Resources = []string{
			"network-policy-ingress.yaml",
			"network-policy-egress.yaml",
		}

		// If declared in dp, create namespace with labels and annotations
		for _, ns := range d.Spec.DeploymentPackageRef.Namespaces {
			nsBundleName := fmt.Sprintf("%s-%s", ns.Name, initNsBundleName)
			err = injectNamespaceToSubDir(ns, fleetPath, nsBundleName)
			if err != nil {
				return err
			}

			// Have main fleet.yaml wait until namespace resource is created
			item := DependsOnItem{
				Name: nsBundleName,
			}
			fleetConf.DependsOn = append(fleetConf.DependsOn, item)
		}

		if hasPreHook {
			// If pre hook present with image pull then generate
			// secret in separate bundle.
			err := injectImageCredentialSecretToSubDir(
				app.HelmApp.ImageRegistry,
				app.HelmApp.ImageRegistrySecretName,
				d.Namespace, namespace, kc, bundleName, fleetPath)
			if err != nil {
				return err
			}
			// Have main fleet.yaml wait until namespace resource is created
			secretBundlename := "pre-install-secret-" + bundleName
			item := DependsOnItem{
				Name: secretBundlename,
			}
			fleetConf.DependsOn = append(fleetConf.DependsOn, item)
		} else if hasImageCreds {
			// Generate kutomization for image credential
			err := injectImageCredentialToKustomization(
				app.HelmApp.ImageRegistry,
				app.HelmApp.ImageRegistrySecretName,
				d.Namespace, namespace, kc, bundleName, k)
			if err != nil {
				return err
			}
		}

		err = WriteKustomization(filepath.Join(fleetPath, "kustomize"), k)
		if err != nil {
			return err
		}
		fleetConf.Kustomize.Dir = "./kustomize"

		// Generate fleet.yaml from Config
		err = WriteFleetConfig(fleetPath, fleetConf)
		if err != nil {
			return err
		}

	}

	return nil
}

func newFleetConfig(appName string, appMap map[string]v1beta1.Application, depID string, depName string, generation int64, namespace string) Config {
	app := appMap[appName]
	bundlename := BundleName(app, depName)

	repo := app.HelmApp.Repo
	u, err := url.Parse(repo)
	if err != nil {
		return Config{}
	}

	repoURL := app.HelmApp.Repo
	chartURL := app.HelmApp.Chart
	if u.Scheme == "oci" {
		repoURL = ""
		chartURL, _ = url.JoinPath(app.HelmApp.Repo, app.HelmApp.Chart)
	}

	deleteCRDResources := utils.DeleteCRDResources()

	// Create Config with the given application information
	fleetConf := Config{
		Name:               bundlename,
		DefaultNamespace:   namespace,
		DeleteCRDResources: deleteCRDResources,
		NamespaceLabels:    make(map[string]string),
		Labels: DeployLabels{
			AppName:              app.Name,
			BundleType:           BundleTypeApp.String(),
			DeploymentID:         depID,
			DeploymentGeneration: fmt.Sprint(generation),
		},
		Helm: HelmApp{
			ReleaseName: bundlename,
			Repo:        repoURL,
			Chart:       chartURL,
			Version:     app.HelmApp.Version,
			ValuesFiles: []string{},
		},
	}

	// Set depends on bundle names, all dependencies must exist in the same deployment package
	fleetConf.DependsOn = []DependsOnItem{}
	for _, dep := range app.DependsOn {
		depApp := appMap[dep]
		depBundle := BundleName(depApp, depName)
		item := DependsOnItem{
			Name: depBundle,
		}
		fleetConf.DependsOn = append(fleetConf.DependsOn, item)
	}

	// Create diff configuration for IgnoreResources, currently only ConfigMap is supported
	fleetConf.Diff = &DiffOptions{}
	for _, res := range app.IgnoreResources {
		ns := app.Namespace
		if res.Namespace != "" {
			ns = res.Namespace
		}

		switch res.Kind {
		//Check for case ValidatingWebhookConfiguration
		case "ConfigMap":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "v1",
				Name:       res.Name,
				Namespace:  ns,
				Operations: IgnoreConfigMapOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "ValidatingWebhookConfiguration":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "admissionregistration.k8s.io/v1",
				Name:       res.Name,
				Operations: IgnoreVWCOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "MutatingWebhookConfiguration":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "admissionregistration.k8s.io/v1",
				Name:       res.Name,
				Operations: IgnoreMWCOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "Secret":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "v1",
				Name:       res.Name,
				Namespace:  ns,
				Operations: IgnoreSecretOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "CustomResourceDefinition":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "apiextensions.k8s.io/v1",
				Name:       res.Name,
				Operations: IgnoreCRDOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "EnvoyFilter":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "networking.istio.io/v1beta1",
				Name:       res.Name,
				Namespace:  ns,
				Operations: IgnoreEnvoyOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "Deployment":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "apps/v1",
				Name:       res.Name,
				Namespace:  ns,
				Operations: IgnoreDeploymentOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		case "Job":
			patch := ComparePatch{
				Kind:       res.Kind,
				APIVersion: "batch/v1",
				Name:       res.Name,
				Namespace:  ns,
				Operations: IgnoreJobOp,
			}
			fleetConf.Diff.ComparePatches = append(fleetConf.Diff.ComparePatches, patch)
		default:
		}
	}

	return fleetConf
}

func valuesFromSecret(secretName string, namespace string, kc client.Client) (string, error) {
	// Secret is not set, return early
	if secretName == "" {
		return "", nil
	}

	values := &corev1.Secret{}
	if err := kc.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}, values); err != nil {
		return "", err
	}

	return string(values.Data["values"]), nil
}

func BundleName(app v1beta1.Application, depName string) string {
	token := app.Name
	if app.RedeployAfterUpdate {
		token = token + app.Version
	}
	token = token + depName

	hash := sha1.New() // #nosec G401 todo: update it
	_, err := hash.Write([]byte(token))
	if err != nil {
		u := uuid.New()
		n := fmt.Sprintf("bw-%s", u.String())
		log.Warnf("failed to create bundle name for deployment name %s - alternative bundle name is %s", depName, n)
		return n
	}

	hashInHex := hex.EncodeToString(hash.Sum(nil))

	// Generate a UUID using the SHA-1 hash as a namespace
	u := uuid.NewSHA1(uuid.Nil, []byte(hashInHex))
	uuidStr := u.String()

	// Extract truncatedUUIDLength characters from the UUID
	bundleName := fmt.Sprintf("b-%s", uuidStr[:truncatedUUIDLength])

	return bundleName
}

// injectNamespaceToSubDir adds namespace with required labels and annotations to new fleet.yaml.
func injectNamespaceToSubDir(ns v1beta1.Namespace, fleetPath string, bundleName string) error {
	subFleetConf := Config{
		Name:                 bundleName,
		DefaultNamespace:     ns.Name,
		NamespaceLabels:      ns.Labels,
		NamespaceAnnotations: ns.Annotations,
	}

	// Generate fleet.yaml from Config
	err := WriteFleetConfig(filepath.Join(fleetPath, ns.Name+"-ns"), subFleetConf)
	if err != nil {
		return err
	}

	// Adding an empty YAML file to trigger bundle creation
	emptyYaml := map[string]interface{}{}
	err = WriteResourceConfig(
		filepath.Join(fleetPath, ns.Name+"-ns"),
		emptyYaml, "empty.yaml")
	if err != nil {
		return err
	}

	return nil
}

// injectImageCredentialToKustomization extracts image credentials from the
// image registry secret and generate Kustomization configuration.
func injectImageCredentialToKustomization(reg string, regSecret string, secretNamespace string,
	appNamespace string, kc client.Client, bundleName string, k *kustomize.Kustomization) error {
	creds := &corev1.Secret{}
	if err := kc.Get(context.Background(), client.ObjectKey{
		Namespace: secretNamespace,
		Name:      regSecret,
	}, creds); err != nil {
		return err
	}

	credmap := map[string]interface{}{
		"auths": map[string]interface{}{
			reg: map[string]string{
				"username": string(creds.Data["username"]),
				"password": string(creds.Data["password"]),
			},
		},
	}
	credjson, err := json.Marshal(credmap)
	if err != nil {
		return err
	}
	contents := fmt.Sprintf(`.dockerconfigjson=%s`, credjson)
	secretGenerator := []kustomize.SecretArgs{
		{
			GeneratorArgs: kustomize.GeneratorArgs{
				Namespace: appNamespace,
				Name:      bundleName,
				KvPairSources: kustomize.KvPairSources{
					LiteralSources: []string{
						contents,
						fmt.Sprintf("accessKeyId=%s", creds.Data["username"]),
						fmt.Sprintf("secretKey=%s", creds.Data["password"]),
					},
				},
			},
			Type: "kubernetes.io/dockerconfigjson",
		},
	}
	generatorOption := &kustomize.GeneratorOptions{
		DisableNameSuffixHash: true,
	}

	k.SecretGenerator = secretGenerator
	k.GeneratorOptions = generatorOption

	return nil
}

// injectImageCredentialSecretToSubDir extracts image credentials from the
// image registry secret and generates secret with new fleet.yaml.
func injectImageCredentialSecretToSubDir(reg string, regSecret string,
	secretNamespace string, appNamespace string, kc client.Client,
	bundleName string, fleetPath string) error {
	creds := &corev1.Secret{}
	if err := kc.Get(context.Background(), client.ObjectKey{
		Namespace: secretNamespace,
		Name:      regSecret,
	}, creds); err != nil {
		return err
	}

	newSecret := &SecretSpec{
		Kind:       "Secret",
		APIVersion: "v1",
		Metadata: Metadata{
			Name:      bundleName,
			Namespace: appNamespace,
		},
		Type: string(corev1.SecretTypeDockerConfigJson),
	}

	credmap := map[string]interface{}{
		"auths": map[string]interface{}{
			reg: map[string]string{
				"username": string(creds.Data["username"]),
				"password": string(creds.Data["password"]),
			},
		},
	}

	credjson, err := json.Marshal(credmap)
	if err != nil {
		return err
	}

	secretBundlename := "pre-install-secret-" + bundleName
	subFleetConf := Config{
		Name:             secretBundlename,
		DefaultNamespace: appNamespace,
	}

	// Generate fleet.yaml from Config
	err = WriteFleetConfig(filepath.Join(fleetPath, "secret-dir"), subFleetConf)
	if err != nil {
		return err
	}

	newSecret.Data = make(map[string][]byte)
	newSecret.Data[".dockerconfigjson"] = credjson
	newSecret.Data["accessKeyId"] = creds.Data["username"]
	newSecret.Data["secretKey"] = creds.Data["password"]

	// Write resource file
	err = WriteResourceConfig(
		filepath.Join(fleetPath, "secret-dir"),
		newSecret, "image-reg-secret.yaml")
	if err != nil {
		return err
	}

	return nil
}

func WriteFleetConfig(basedir string, fleetconfig Config) error {
	data, err := yaml.Marshal(&fleetconfig)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, "fleet.yaml", data)
}

func WriteResourceConfig(basedir string, v interface{}, name string) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, name, data)
}

func WriteNetworkPolicyConfig(basedir string, nwPolicy NetworkPolicy, name string) error {
	data, err := yaml.Marshal(&nwPolicy)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, name, data)
}

func WriteKustomization(basedir string, k *kustomize.Kustomization) error {
	data, err := yaml.Marshal(&k)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, "kustomization.yaml", data)
}

func WriteExtraValues(basedir string, filename string, e *ExtraValues) error {
	data, err := yaml.Marshal(&e)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, filename, data)
}

func getRegistryProjectName(d *v1beta1.Deployment, nexusClient nexus.RuntimeProjectsGetter) (string, error) {
	projectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
	if projectID == "" {
		return "", fmt.Errorf("project-id not found in deployment labels")
	}
	runtimeProjects, err := nexusClient.RuntimeProjects().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Failed to List Nexus Runtime Project %v", err)
		return "", err
	}
	for _, runtimeProject := range runtimeProjects.Items {
		if runtimeProject.GetUID() == types.UID(projectID) {
			log.Debugf("Found Nexus project %s with UID %s", runtimeProject.GetName(), runtimeProject.UID)
			projectName := runtimeProject.DisplayName()
			orgName := runtimeProject.GetLabels()[NexusOrgLabel]
			if orgName == "" {
				return "", fmt.Errorf("nexus project %s has no label %s", projectName, NexusOrgLabel)
			}

			return fmt.Sprintf("%s-%s-%s", RegistryProjectPrefix, orgName, projectName), nil
		}
	}

	return "", fmt.Errorf("unable to find nexus runtime project with UID: %s", projectID)
}
