// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package randomtoken

import (
	"crypto/rand"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/undefinedlabs/go-mpatch"
	"io"
	"math/big"
	"testing"
)

func unpatchAll(list []*mpatch.Patch) error {
	for _, p := range list {
		err := p.Unpatch()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestGenerate(t *testing.T) {
	tk, err := Generate()
	assert.NoError(t, err)
	assert.NotEqual(t, "", tk)
}

func TestGenerate_ErrorRand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		f1, err := mpatch.PatchMethod(rand.Int, func(rand io.Reader, max *big.Int) (n *big.Int, err error) {
			return nil, errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error: %v", err)
		}

		return []*mpatch.Patch{f1}
	}
	pList := patch(ctrl)
	tk, err := Generate()
	assert.Error(t, err)
	assert.Equal(t, "", tk)
	err = unpatchAll(pList)
	if err != nil {
		t.Error(err)
	}
}
