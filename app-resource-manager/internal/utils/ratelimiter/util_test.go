// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ratelimiter

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetRateLimiterParams(t *testing.T) {
	err := os.Setenv(rateLimiterQPS, "20")
	assert.NoError(t, err)
	err = os.Setenv(rateLimiterBurst, "300")
	assert.NoError(t, err)
	qps, burst, err := GetRateLimiterParams()
	assert.NoError(t, err)

	assert.Equal(t, float64(20), qps)
	assert.Equal(t, float64(300), float64(burst))

}

func TestGetRateLimiterParamsQpsError(t *testing.T) {
	err := os.Setenv(rateLimiterQPS, "test-error")
	assert.NoError(t, err)

	qps, burst, err := GetRateLimiterParams()
	assert.Error(t, err)
	assert.Equal(t, 0, int(qps))
	assert.Equal(t, 0, int(burst))

}

func TestGetRateLimiterParamsBurstParamError(t *testing.T) {
	err := os.Setenv(rateLimiterQPS, "20")
	assert.NoError(t, err)
	err = os.Setenv(rateLimiterBurst, "test-error")
	assert.NoError(t, err)
	qps, burst, err := GetRateLimiterParams()
	assert.Error(t, err)
	assert.Equal(t, 0, int(qps))
	assert.Equal(t, 0, int(burst))

}
