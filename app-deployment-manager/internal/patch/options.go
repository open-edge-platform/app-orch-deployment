// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package patch

// Option is some configuration that modifies options for a patch request.
type Option interface {
	ApplyToHelper(*HelperOptions)
}

type HelperOptions struct {
	// IncludeStatusObservedGeneration sets the status.observedGeneration field
	// on the incoming object to match metadata.generation, only if there is a change.
	IncludeStatusObservedGeneration bool
}

type WithStatusObservedGeneration struct{}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithStatusObservedGeneration) ApplyToHelper(in *HelperOptions) {
	in.IncludeStatusObservedGeneration = true
}
