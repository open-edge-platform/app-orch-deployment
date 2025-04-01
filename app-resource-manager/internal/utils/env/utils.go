// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"
	"strconv"
)

const (
	msgSizeLimit = "MSG_SIZE_LIMIT"
)

// GetMessageSizeLimit gets message size limit
func GetMessageSizeLimit() (int64, error) {
	msgSizeLimitStr := os.Getenv(msgSizeLimit)
	msgSizeLimit, err := strconv.ParseInt(msgSizeLimitStr, 10, 64)
	if err != nil {
		return 0, err
	}
	msgSizeLimitBytes := msgSizeLimit * 1024 * 1024
	return msgSizeLimitBytes, nil
}
