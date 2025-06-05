// SPDX-FileCopyrightText: (C) 2025-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CloneRepository clones a git repository to a temporary directory
func CloneRepository(repoURL, branch string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "repo-clone-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Execute git clone command
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, repoURL, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir) // Clean up directory on failure
		return "", fmt.Errorf("git clone failed: %w", err)
	}

	return tempDir, nil
}

// CloneCirrosVM clones the cirros-vm repository and returns the path to the repository
func CloneCirrosVM() (string, error) {
	repoURL := "https://github.com/open-edge-platform/app-orch-catalog.git"
	branch := "main"

	tempDir, err := CloneRepository(repoURL, branch)
	if err != nil {
		return "", err
	}

	// The specific path to the cirros-vm directory within the cloned repository
	cirrosVMPath := filepath.Join(tempDir, "app-orch-tutorials", "cirros-vm")

	// Verify that the directory exists
	if _, err := os.Stat(cirrosVMPath); os.IsNotExist(err) {
		os.RemoveAll(tempDir) // Clean up
		return "", fmt.Errorf("cirros-vm directory not found in cloned repository")
	}

	return cirrosVMPath, nil
}
