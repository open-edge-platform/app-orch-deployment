// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// todo: run below command when running make generate-mocks
// run it: mockery --name VaultAuth --filename mockery_vaultauth.go --structname MockeryVaultAuth --srcpkg=github.com/open-edge-platform/orch-library/go/pkg/auth

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// VaultAuth is an autogenerated mock type for the VaultAuth type
type VaultAuth struct {
	mock.Mock
}

// CreateClientSecret provides a mock function with given fields: ctx, username, password
func (_m *VaultAuth) CreateClientSecret(ctx context.Context, username string, password string) (string, error) {
	ret := _m.Called(ctx, username, password)

	if len(ret) == 0 {
		panic("no return value specified for CreateClientSecret")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, error)); ok {
		return rf(ctx, username, password)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, username, password)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, username, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetM2MToken provides a mock function with given fields: ctx
func (_m *VaultAuth) GetM2MToken(ctx context.Context) (string, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetM2MToken")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (string, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVaultToken provides a mock function with given fields: ctx
func (_m *VaultAuth) GetVaultToken(ctx context.Context) (string, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetVaultToken")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (string, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Logout provides a mock function with given fields: ctx
func (_m *VaultAuth) Logout(_ context.Context) error {
	//ret := _m.Called(ctx)
	//
	//if len(ret) == 0 {
	//	panic("no return value specified for Logout")
	//}
	//
	//var r0 error
	//if rf, ok := ret.Get(0).(func(context.Context) error); ok {
	//	r0 = rf(ctx)
	//} else {
	//	r0 = ret.Error(0)
	//}
	//
	//return r0

	// make sure all logout function returns no error for testing
	// todo: uncomment above lines once we have new test case for logout
	return nil
}

// NewVaultAuth creates a new instance of VaultAuth. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewVaultAuth(t interface {
	mock.TestingT
	Cleanup(func())
}) *VaultAuth {
	mock := &VaultAuth{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
