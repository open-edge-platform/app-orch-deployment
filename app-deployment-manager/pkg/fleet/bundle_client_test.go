// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/undefinedlabs/go-mpatch"
	"k8s.io/client-go/rest"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
)

func TestNewBundleClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		restClientFor, err := mpatch.PatchMethod(rest.RESTClientFor, func(config *rest.Config) (*rest.RESTClient, error) {
			return &rest.RESTClient{}, nil

		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, restClientFor}
	}

	pList := patch(ctrl)
	cs, err := NewClusterClient("")
	assert.NoError(t, err)
	assert.NotNil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestNewBundleClient_EmptyKubeConfig(t *testing.T) {
	// failed scenario with wrong kubeconfig
	_, err := NewBundleClient("")
	assert.Error(t, err)
}

func TestNewBundleClient_FailedCreatingRestConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return nil, errors.New("temp error")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		restClientFor, err := mpatch.PatchMethod(rest.RESTClientFor, func(config *rest.Config) (*rest.RESTClient, error) {
			return &rest.RESTClient{}, nil

		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, restClientFor}
	}

	pList := patch(ctrl)
	cs, err := NewClusterClient("")
	assert.Error(t, err)
	assert.Nil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestNewBundleClient_FailedCreatingNewClientSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		restClientFor, err := mpatch.PatchMethod(rest.RESTClientFor, func(config *rest.Config) (*rest.RESTClient, error) {
			return nil, errors.New("temp error")

		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, restClientFor}
	}

	pList := patch(ctrl)
	cs, err := NewClusterClient("")
	assert.Error(t, err)
	assert.Nil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}
