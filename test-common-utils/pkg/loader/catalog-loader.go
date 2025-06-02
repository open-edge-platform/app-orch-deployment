// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/auth"
	"github.com/open-edge-platform/app-orch-deployment/test-common-utils/pkg/git"
	catalogloader "github.com/open-edge-platform/orch-library/go/pkg/loader"
	"os"
	"path/filepath"
	"strconv"
)

func Upload(paths []string) error {
	autoCert, err := strconv.ParseBool(os.Getenv("AUTO_CERT"))
	orchDomain := os.Getenv("ORCH_DOMAIN")
	if err != nil || !autoCert || orchDomain == "" {
		orchDomain = "kind.internal"
	}

	orchProject := "sample-project"
	if orchProjectEnv := os.Getenv("ORCH_PROJECT"); orchProjectEnv != "" {
		orchProject = orchProjectEnv
	}

	err = UploadFiles(paths, orchDomain, orchProject)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	fmt.Println("Apps Uploaded 😊")
	return nil
}

func UploadFiles(paths []string, domain string, projectName string) error {
	apiBaseURL := "https://api." + domain
	keycloakServer := fmt.Sprintf("keycloak.%s", domain)

	loader := catalogloader.NewLoader(apiBaseURL, projectName)
	token, err := auth.SetUpAccessToken(keycloakServer)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	err = loader.LoadResources(context.Background(), token, paths)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// UploadCirrosVM clones the cirros-vm repository and loads it into the catalog
func UploadCirrosVM() error {
	// Clone the repository and get the path to cirros-vm
	cirrosVMPath, err := git.CloneCirrosVM()
	if err != nil {
		return fmt.Errorf("failed to clone cirros-vm repository: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(filepath.Dir(cirrosVMPath))) // Clean up the temporary directory after upload

	// Upload the cirros-vm to the catalog
	err = Upload([]string{cirrosVMPath})
	if err != nil {
		return fmt.Errorf("failed to upload cirros-vm: %w", err)
	}

	return nil
}
