// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package k8sclient

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/undefinedlabs/go-mpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
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

func TestNewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		os.Setenv("RATE_LIMITER_QPS", "20")
		os.Setenv("RATE_LIMITER_BURST", "1000")
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		newForConfig, err := mpatch.PatchMethod(kubernetes.NewForConfig, func(c *rest.Config) (*kubernetes.Clientset, error) {
			return &kubernetes.Clientset{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, newForConfig}
	}

	pList := patch(ctrl)
	cs, err := NewClient("")
	assert.NoError(t, err)
	assert.NotNil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestNewClient_EmptyKubeConfig(t *testing.T) {
	// failed scenario with wrong kubeconfig
	_, err := NewClient("")
	assert.Error(t, err)
}

func TestNewClient_FailedCreatingRestConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return nil, errors.New("temp error")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		newForConfig, err := mpatch.PatchMethod(kubernetes.NewForConfig, func(c *rest.Config) (*kubernetes.Clientset, error) {
			return &kubernetes.Clientset{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, newForConfig}
	}

	pList := patch(ctrl)
	cs, err := NewClient("")
	assert.Error(t, err)
	assert.Nil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestNewClient_FailedCreatingNewClientSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		createRestConfig, err := mpatch.PatchMethod(utils.CreateRestConfig, func(kubeConfig string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		newForConfig, err := mpatch.PatchMethod(kubernetes.NewForConfig, func(c *rest.Config) (*kubernetes.Clientset, error) {
			return nil, errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{createRestConfig, newForConfig}
	}

	pList := patch(ctrl)
	cs, err := NewClient("")
	assert.Error(t, err)
	assert.Nil(t, cs)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}
