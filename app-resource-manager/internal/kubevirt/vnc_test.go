// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testVNCPath = "/vnc/projectID1/appID1/cluster1/vm1"
)

func TestNewVNCPath(t *testing.T) {
	p, err := newVNCPath(testVNCPath)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewVNCPath_WrongPath(t *testing.T) {
	p, err := newVNCPath("")
	assert.Error(t, err)
	assert.Nil(t, p)
}
