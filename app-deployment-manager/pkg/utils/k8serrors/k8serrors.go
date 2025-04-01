// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package k8serrors

import (
	typederror "github.com/open-edge-platform/orch-library/go/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func K8sToTypedError(err error) error {
	switch errors.ReasonForError(err) {
	case metav1.StatusReasonConflict:
		return typederror.NewConflict(err.Error())
	case metav1.StatusReasonBadRequest:
		return typederror.NewInvalid(err.Error())
	case metav1.StatusReasonInternalError:
		return typederror.NewInternal(err.Error())
	case metav1.StatusReasonAlreadyExists:
		return typederror.NewAlreadyExists(err.Error())
	case metav1.StatusReasonForbidden:
		return typederror.NewForbidden(err.Error())
	case metav1.StatusReasonTimeout:
		return typederror.NewTimeout(err.Error())
	case metav1.StatusReasonInvalid:
		return typederror.NewInvalid(err.Error())
	case metav1.StatusReasonNotFound:
		return typederror.NewNotFound(err.Error())
	case metav1.StatusReasonServerTimeout:
		return typederror.NewTimeout(err.Error())
	case metav1.StatusReasonServiceUnavailable:
		return typederror.NewUnavailable(err.Error())
	case metav1.StatusReasonTooManyRequests:
		return typederror.NewUnavailable(err.Error())
	case metav1.StatusReasonRequestEntityTooLarge:
		return typederror.NewInvalid(err.Error())
	case metav1.StatusReasonUnauthorized:
		return typederror.NewForbidden(err.Error())

	}
	return typederror.NewUnknown(err.Error())
}
