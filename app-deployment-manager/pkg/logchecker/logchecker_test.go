// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logchecker

import (
	"testing"
)

func TestLogChecker_AddCheck(t *testing.T) {
	checker := New()
	checker.AddCheck(`codecommit.*Permission denied`, "Issue detected: CodeCommit permission problem.")
	checker.AddCheck(`error.*fatal`, "Critical error detected in log.")

	if len(checker.patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(checker.patterns))
	}
}

func TestLogChecker_ProcessLog(t *testing.T) {
	checker := New()
	// Add patterns and expected responses
	checker.AddCheck(`codecommit.*Permission denied`, "Issue detected: CodeCommit permission problem.")
	checker.AddCheck(`error.*fatal`, "Critical error detected in log.")

	// Define test cases
	testCases := []struct {
		name     string
		log      string
		expected string
	}{
		{"Match CodeCommit Error", "Failed: codecommit operation, Permission denied", "Issue detected: CodeCommit permission problem."},
		{"Match Fatal Error", "Unexpected error: something fatal occurred", "Critical error detected in log."},
		{"No Match", "Normal operation completed successfully", "Normal operation completed successfully"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.ProcessLog(tc.log)
			if result != tc.expected {
				t.Errorf("Test '%s' failed: expected '%s', got '%s'", tc.name, tc.expected, result)
			}
		})
	}
}
