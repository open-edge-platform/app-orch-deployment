// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ratelimiter

import (
	"github.com/open-edge-platform/orch-library/go/dazl"
	"os"
	"strconv"
)

const (
	rateLimiterQPS   = "RATE_LIMITER_QPS"
	rateLimiterBurst = "RATE_LIMITER_BURST"
)

var log = dazl.GetPackageLogger()

func GetRateLimiterParams() (float64, int64, error) {
	qps := os.Getenv(rateLimiterQPS)
	qpsValue, err := strconv.ParseFloat(qps, 32)
	if err != nil {
		log.Warn(err)
		return 0, 0, err

	}
	burst := os.Getenv(rateLimiterBurst)
	burstValue, err := strconv.ParseInt(burst, 10, 32)
	if err != nil {
		log.Warn(err)
		return 0, 0, err

	}
	return qpsValue, burstValue, nil

}
