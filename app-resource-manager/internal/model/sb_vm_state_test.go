// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVMState_String(t *testing.T) {
	stateUnknown := VMStateUnknown
	assert.Equal(t, "Unknown", stateUnknown.String())
	stateStart := VMStateStart
	assert.Equal(t, "Start", stateStart.String())
	stateStop := VMStateStop
	assert.Equal(t, "Stop", stateStop.String())
	stateRestart := VMStateRestart
	assert.Equal(t, "Restart", stateRestart.String())
}
