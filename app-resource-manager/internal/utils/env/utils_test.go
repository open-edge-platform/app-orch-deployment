// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetMessageSizeLimit(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		expectedSize int64
		expectingErr bool
	}{
		{
			name:         "normal case",
			envVar:       "2",
			expectedSize: 2097152,
			expectingErr: false,
		},
		{
			name:         "no env var",
			envVar:       "",
			expectedSize: 0,
			expectingErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.Setenv(msgSizeLimit, tt.envVar)
			assert.NoError(t, err)

			size, err := GetMessageSizeLimit()

			if tt.expectingErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSize, size)
			}

			err = os.Unsetenv(msgSizeLimit)
			assert.NoError(t, err)
		})
	}
}
