// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package k8serrors

import (
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestK8sToTypedError(t *testing.T) {
	assert.True(t, errors.IsInvalid(K8sToTypedError(k8serrors.NewInvalid(schema.GroupKind{}, "invalid", nil))))
	assert.True(t, errors.IsForbidden(K8sToTypedError(k8serrors.NewForbidden(schema.GroupResource{}, "forbidden", nil))))
	assert.True(t, errors.IsConflict(K8sToTypedError(k8serrors.NewConflict(schema.GroupResource{}, "conflict", nil))))
	assert.True(t, errors.IsInvalid(K8sToTypedError(k8serrors.NewBadRequest("bad request"))))
	assert.True(t, errors.IsAlreadyExists(K8sToTypedError(k8serrors.NewAlreadyExists(schema.GroupResource{}, "alreadyExist"))))
	assert.True(t, errors.IsTimeout(K8sToTypedError(k8serrors.NewTimeoutError("", 2))))
	assert.True(t, errors.IsUnavailable(K8sToTypedError(k8serrors.NewServiceUnavailable("unavailable"))))
	assert.True(t, errors.IsInvalid(K8sToTypedError(k8serrors.NewRequestEntityTooLargeError("too large entity"))))
	assert.True(t, errors.IsForbidden(K8sToTypedError(k8serrors.NewUnauthorized("unauthorized"))))
	assert.True(t, errors.IsUnavailable(K8sToTypedError(k8serrors.NewTooManyRequests("", 2))))
	assert.True(t, errors.IsUnavailable(K8sToTypedError(k8serrors.NewTooManyRequests("", 2))))
	assert.True(t, errors.IsNotFound(K8sToTypedError(k8serrors.NewNotFound(schema.GroupResource{}, "notfound"))))
	assert.True(t, errors.IsInternal(K8sToTypedError(k8serrors.NewInternalError(errors.NewInternal("")))))
	assert.True(t, errors.IsTimeout(K8sToTypedError(k8serrors.NewServerTimeout(schema.GroupResource{}, "", 2))))
	assert.True(t, errors.IsUnknown(K8sToTypedError(errors.NewInternal("not a k8s error"))))

}
