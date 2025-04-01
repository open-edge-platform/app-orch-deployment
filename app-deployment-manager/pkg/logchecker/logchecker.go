// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package logchecker is a library for checking and transforming log messages based on user-defined rules.
package logchecker

import (
	"regexp"
	"sync"
)

type LogChecker struct {
	patterns map[*regexp.Regexp]string
	mu       sync.RWMutex // Mutex for thread-safe writes to the map
}

func New() *LogChecker {
	return &LogChecker{
		patterns: make(map[*regexp.Regexp]string),
	}
}

func (lc *LogChecker) AddCheck(pattern, response string) {
	re := regexp.MustCompile(pattern) // Compile the pattern
	lc.mu.Lock()                      // Ensure thread safety on writes
	lc.patterns[re] = response
	lc.mu.Unlock()
}

func (lc *LogChecker) ProcessLog(log string) string {
	lc.mu.RLock() // Read lock for thread-safe reading
	defer lc.mu.RUnlock()
	for re, response := range lc.patterns {
		if re.MatchString(log) {
			return response
		}
	}
	return log // Return the original log if no patterns match
}
