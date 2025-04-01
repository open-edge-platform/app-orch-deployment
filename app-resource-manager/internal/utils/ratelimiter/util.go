// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ratelimiter

import (
	"os"
	"strconv"
)

const (
	rateLimiterQPS   = "RATE_LIMITER_QPS"
	rateLimiterBurst = "RATE_LIMITER_BURST"
)

func GetRateLimiterParams() (float64, int64, error) {
	qps := os.Getenv(rateLimiterQPS)
	qpsValue, err := strconv.ParseFloat(qps, 32)
	if err != nil {
		return 0, 0, err
	}
	burst := os.Getenv(rateLimiterBurst)
	burstValue, err := strconv.ParseInt(burst, 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return qpsValue, burstValue, nil
}
