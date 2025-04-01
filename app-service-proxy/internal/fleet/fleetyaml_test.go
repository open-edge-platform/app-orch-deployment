// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDefaultNamespace = "orchestrator-system"
	testGitRepoName      = "repo"
	testAgentChart       = "oci://test.com/app-service-proxy-agent"
	testAgentVersion     = "0.1.0"
	testToken            = "token"
	testTTLHours         = "100"
	testUpdatedTime      = "2006-01-02 15:04:05"
)

func TestGetFleetYAML(t *testing.T) {
	dir := t.TempDir()
	y, err := GetFleetYAML(dir, testDefaultNamespace, testGitRepoName, testAgentChart, testAgentVersion)
	assert.NoError(t, err)
	assert.NotNil(t, y)
	assert.Equal(t, testDefaultNamespace, y.DefaultNamespace)
	assert.Equal(t, testGitRepoName, y.Helm.ReleaseName)
	assert.Equal(t, testAgentChart, y.Helm.Chart)
	assert.Equal(t, testAgentVersion, y.Helm.Version)
}

func TestGetKustomizationYAML(t *testing.T) {
	k := GetKustomizationYAML()
	assert.NotNil(t, k)
	assert.Equal(t, 0, len(k.Resources))
}

func TestWriteFleetYAML(t *testing.T) {
	dir := t.TempDir()
	y, err := GetFleetYAML(dir, testDefaultNamespace, testGitRepoName, testAgentChart, testAgentVersion)
	assert.NoError(t, err)
	assert.NotNil(t, y)
	err = WriteFleetYAML(dir, y)
	assert.NoError(t, err)

	// error case
	err = WriteFleetYAML(dir, nil)
	assert.NoError(t, err)
}

func TestWriteKustomization(t *testing.T) {
	dir := t.TempDir()
	k := GetKustomizationYAML()
	assert.NotNil(t, k)
	assert.Equal(t, 0, len(k.Resources))
	assert.NotNil(t, k)
	err := WriteKustomization(dir, k)
	assert.NoError(t, err)
}

func TestWriteSecretToken(t *testing.T) {
	dir := t.TempDir() // Create a temporary directory for the test
	tcName := "kust"   // The test case name or directory name under overlays

	// Expected file path based on the function logic
	expectedFilePath := filepath.Join(dir, FleetOverlaysDirName, tcName, TokenSecretFileName)

	// Call WriteSecretToken, which is expected to succeed
	err := WriteSecretToken(dir, tcName, testDefaultNamespace, testToken, testTTLHours, testUpdatedTime)
	assert.NoError(t, err)

	// Check if the expected file was created
	_, err = os.Stat(expectedFilePath)
	assert.NoError(t, err, "The file should have been created by WriteSecretToken")
}

func TestReadFleetYAMLFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, FleetYamlFileName)
	invalidYAMLContent := ":\n"

	err := os.WriteFile(filePath, []byte(invalidYAMLContent), 0644) // #nosec G306
	assert.NoError(t, err, "Setup should not fail")

	_, err = ReadFleetYAMLFile(dir)
	assert.Error(t, err, "Should return error for invalid YAML")
}

func TestWriteFleetYAMLPermissions(t *testing.T) {
	dir := t.TempDir()
	y, _ := GetFleetYAML(dir, testDefaultNamespace, testGitRepoName, testAgentChart, testAgentVersion)

	// Write the Fleet YAML file
	err := WriteFleetYAML(dir, y)
	assert.NoError(t, err)

	// Get file info
	fileInfo, err := os.Stat(filepath.Join(dir, FleetYamlFileName))
	assert.NoError(t, err)

	// Isolate the permission bits
	permissions := fileInfo.Mode().Perm()

	// Assert that permissions match '0600'
	assert.Equal(t, os.FileMode(0600), permissions, "The fleet.yaml file permissions should be restrictive")
}

func TestWriteFleetYAMLInvalidDir(t *testing.T) {
	// Create a temporary file, which ensures the path exists but is not a directory
	tempFile, err := os.CreateTemp("", "fleet")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()                 // Close the file, but keep the path
	defer os.Remove(tempFile.Name()) // Clean up after the test

	// Try to write the Fleet YAML to this path (which is a file, not a directory)
	y, _ := GetFleetYAML("", testDefaultNamespace, testGitRepoName, testAgentChart, testAgentVersion)
	err = WriteFleetYAML(tempFile.Name(), y)

	// Verify that the function returns an error as expected
	assert.Error(t, err, "Should return error for invalid directory")
}

func TestGetFleetYAMLWithPreExistingIncompleteFleetYAML(t *testing.T) {
	dir := t.TempDir()

	// Create an incomplete fleet.yaml file in the temp directory
	initialContent := `name: incomplete-repo`
	err := os.WriteFile(filepath.Join(dir, FleetYamlFileName), []byte(initialContent), 0644) // #nosec G306
	assert.NoError(t, err)

	// Now, try to get the FleetYAML struct, which should update the incomplete file
	y, err := GetFleetYAML(dir, testDefaultNamespace, testGitRepoName, testAgentChart, testAgentVersion)
	assert.NoError(t, err)
	assert.NotNil(t, y)
	assert.Equal(t, testGitRepoName, y.Name, "Expected GetFleetYAML to update the name from the file")
}
