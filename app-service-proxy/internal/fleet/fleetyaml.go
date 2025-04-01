// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/vault"
	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"
)

const (
	FleetYamlFileName     = "fleet.yaml"
	FleetValuesFileName   = "values.yaml"
	FleetOverlaysDirName  = "overlays"
	KustomizationFileName = "kustomization.yaml"
	TokenSecretName       = "app-service-proxy-agent-token" // #nosec G101
	TokenSecretFileName   = TokenSecretName + ".yaml"

	TokenKey            = vault.TokenKey
	TTLHoursKey         = vault.TTLHoursKey
	TokenUpdatedTimeKey = "updatedTime"
)

// TODO: in the latest Fleet API repo, FleetYaml struct is defined; once ADM bump up Fleet version, remove this and use Fleet API directly

// FleetYAML is the top-level structure of the fleet.yaml file.
// The fleet.yaml file adds options to a bundle. Any directory with a
// fleet.yaml is automatically turned into a bundle.
// It is copied from Fleet API. TODO: remove after ADM has new Fleet API
// nolint
type FleetYAML struct {
	// Name of the bundle which will be created.
	Name string `json:"name,omitempty"`
	// Labels are copied to the bundle and can be used in a
	// dependsOn.selector.
	Labels map[string]string `json:"labels,omitempty"`

	v1alpha1.BundleSpec
	// TargetCustomizations are used to determine how resources should be
	// modified per target. Targets are evaluated in order and the first
	// one to match a cluster is used for that cluster.
	TargetCustomizations []v1alpha1.BundleTarget `json:"targetCustomizations,omitempty"`
	// ImageScans are optional and used to update container image
	// references in the git repo.
	ImageScans []ImageScanYAML `json:"imageScans,omitempty"`
	// OverrideTargets overrides targets that are defined in the GitRepo
	// resource. If overrideTargets is provided the bundle will not inherit
	// targets from the GitRepo.
	OverrideTargets []v1alpha1.GitTarget `json:"overrideTargets,omitempty"`
}

// ImageScanYAML is a single entry in the ImageScan list from fleet.yaml.
// It is copied from Fleet API. TODO: remove after ADM has new Fleet API
type ImageScanYAML struct {
	// Name of the image scan. Unused.
	Name string `json:"name,omitempty"`
	v1alpha1.ImageScanSpec
}

func GetTargetCustomizationName(clusterID string) string {
	return fmt.Sprintf("app-service-proxy-agent-%s", clusterID)
}

func ReadFleetYAMLFile(basedir string) (*FleetYAML, error) {
	fleetConfig := &FleetYAML{}
	fleetYAMLPath := filepath.Join(basedir, FleetYamlFileName)
	data, err := os.ReadFile(fleetYAMLPath)
	if err != nil {
		return nil, err
	}

	var jsonData interface{}
	err = yaml.Unmarshal(data, &jsonData)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonBytes, fleetConfig)
	if err != nil {
		return nil, err
	}

	return fleetConfig, nil
}

func GetFleetYAML(basedir string, defaultNamespace string, gitRepoName string, agentChart string, agentVersion string) (*FleetYAML, error) {
	// Try to read fleet.yaml file
	fleetConfig, err := ReadFleetYAMLFile(basedir)

	// Failed to fetch fleet.yaml from local git repository
	if err != nil {
		// If fleet.yaml does not exists, define new one
		if os.IsNotExist(err) {
			fleetConfig = &FleetYAML{}
		} else {
			return nil, err
		}
	}

	// Always update fleet.yaml with the latest config
	fleetConfig.DefaultNamespace = defaultNamespace
	fleetConfig.Name = gitRepoName
	fleetConfig.Helm = &v1alpha1.HelmOptions{
		ReleaseName: gitRepoName,
		Chart:       agentChart,
		Version:     agentVersion,
		ValuesFiles: []string{FleetValuesFileName},
	}

	// if targetCustomizations slice is empty, initialize the slice
	if len(fleetConfig.TargetCustomizations) == 0 {
		fleetConfig.TargetCustomizations = make([]v1alpha1.BundleTarget, 0)
	}

	return fleetConfig, nil
}

func GetKustomizationYAML() *types.Kustomization {
	return &types.Kustomization{
		Resources: make([]string, 0),
	}
}

func WriteFleetYAML(basedir string, fleetYAML *FleetYAML) error {
	jsonData, err := json.Marshal(fleetYAML)
	if err != nil {
		return err
	}

	var yamlData interface{}
	err = yaml.Unmarshal(jsonData, &yamlData)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, FleetYamlFileName, data)
}

func WriteKustomization(basedir string, kustomization *types.Kustomization) error {
	data, err := yaml.Marshal(&kustomization)
	if err != nil {
		return err
	}
	return utils.WriteFile(basedir, KustomizationFileName, data)
}

func WriteSecretToken(basedir string, tcName string, namespace string, token string, ttlHours string, tokenUpdatedTime string) error {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TokenSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			TokenKey:            []byte(token),
			TTLHoursKey:         []byte(ttlHours),
			TokenUpdatedTimeKey: []byte(tokenUpdatedTime),
		},
		Type: v1.SecretTypeOpaque,
	}

	jsonData, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	var yamlData interface{}
	err = yaml.Unmarshal(jsonData, &yamlData)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}

	return utils.WriteFile(filepath.Join(basedir, FleetOverlaysDirName, tcName), TokenSecretFileName, data)
}
