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
	"os/exec"
	"path/filepath"
	"time"
)

func Upload(paths []string) error {
	orchDomain := auth.GetOrchDomain()
	orchProject := "sample-project"
	if orchProjectEnv := os.Getenv("ORCH_PROJECT"); orchProjectEnv != "" {
		orchProject = orchProjectEnv
	}

	err := UploadFiles(paths, orchDomain, orchProject)
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

func UploadHttpbinHelm(path, harborPwd string) error {
	chartPath := path + "/helm"                      // Path to your chart directory
	registry := "registry-oci.kind.internal"         // OCI registry URL (without oci:// prefix)
	repo := "catalog-apps-sample-org-sample-project" // Repository name in your OCI registry
	username := "sample-project-edge-mgr"

	// 1. Login to the OCI registry
	regRef := fmt.Sprintf("https://%s/%s", registry, repo)
	loginCmd := exec.Command(
		"helm", "registry", "login",
		"-u", username,
		"--password", harborPwd,
		regRef,
	)
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr
	fmt.Println("Logging in to OCI registry...")
	if err := loginCmd.Run(); err != nil {
		fmt.Printf("Failed to login to OCI registry: %v\n", err)
		os.Exit(1)
	}

	// 2. Package the Helm chart
	version := "0.1.8"
	pkgCmd := exec.Command("helm", "package", chartPath, "--version", version)
	pkgCmd.Stdout = os.Stdout
	pkgCmd.Stderr = os.Stderr
	fmt.Println("Packaging chart...")
	if err := pkgCmd.Run(); err != nil {
		fmt.Printf("Failed to package chart: %v\n", err)
		os.Exit(1)
	}

	chartName := "httpbin"
	chartTGZ := fmt.Sprintf("%s-%s.tgz", chartName, version)

	// 3. Push the chart to OCI registry
	ociRef := fmt.Sprintf("oci://%s/%s", registry, repo)
	pushCmd := exec.Command("helm", "push", chartTGZ, ociRef)
	//pushCmd.Stdout = os.Stdout
	//pushCmd.Stderr = os.Stderr
	fmt.Println("Pushing chart to OCI registry...")
	retries := 40
	var err error
	for i := 0; i < retries; i++ {
		pushCmd = exec.Command("helm", "push", chartTGZ, ociRef)
		if err = pushCmd.Run(); err != nil {
			fmt.Printf("retry count %d\n", i)
			time.Sleep(1 * time.Second)
		}
	}

	if err != nil {
		fmt.Printf("Failed to push chart: %v\n", err)
		os.Exit(1)
	}
	// Optional: Cleanup the packaged file
	os.Remove(filepath.Join(".", chartTGZ))
	fmt.Println("Done!")
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

// UploadHttpbin clones the httpbin repository and loads it into the catalog
func UploadHttpbin() error {
	// Clone the repository and get the path to cirros-vm
	httpBinPath, err := git.CloneHttpbin()
	if err != nil {
		return fmt.Errorf("failed to clone httpbin repository: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(filepath.Dir(httpBinPath))) // Clean up the temporary directory after upload

	// Upload the cirros-vm to the catalog
	err = Upload([]string{httpBinPath})
	if err != nil {
		return fmt.Errorf("failed to upload httpbin: %w", err)
	}

	return nil
}
