// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package patch

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type patchType string

func (p patchType) Key() string {
	return strings.Split(string(p), ".")[0]
}

const (
	specPatch   patchType = "spec"
	statusPatch patchType = "status"
)

var (
	preserveUnstructuredKeys = map[string]bool{
		"kind":       true,
		"apiVersion": true,
		"metadata":   true,
	}
)

type Helper struct {
	client       client.Client
	gvk          schema.GroupVersionKind
	beforeObject client.Object
	before       *unstructured.Unstructured
	after        *unstructured.Unstructured
	changes      map[string]bool
}

// NewHelper returns an initialized Helper.
func NewPatchHelper(obj client.Object, crClient client.Client) (*Helper, error) {
	if isNil(obj) {
		return nil, errors.New("helper could not be created: object is nil")
	}

	gvk, err := apiutil.GVKForObject(obj, crClient.Scheme())
	if err != nil {
		return nil, err
	}

	// Convert the object to unstructured to compare against our before copy.
	unstructuredObj, err := toUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &Helper{
		client:       crClient,
		gvk:          gvk,
		before:       unstructuredObj,
		beforeObject: obj.DeepCopyObject().(client.Object),
	}, nil
}

// Patch will attempt to patch the given object, including its status.
func (h *Helper) Patch(ctx context.Context, obj client.Object, opts ...Option) error {
	if isNil(obj) {
		return errors.New("Patch could not be completed: object is nil")
	}

	gvk, err := apiutil.GVKForObject(obj, h.client.Scheme())
	if err != nil {
		return err
	}
	if gvk != h.gvk {
		return errors.Errorf("unmatched GroupVersionKind, expected %q got %q", h.gvk, gvk)
	}

	// Calculate the options.
	options := &HelperOptions{}
	for _, opt := range opts {
		opt.ApplyToHelper(options)
	}

	// Convert the object to unstructured to compare against the before copy.
	h.after, err = toUnstructured(obj)
	if err != nil {
		return err
	}

	// Determine if the object has status.
	if unstructuredHasStatus(h.after) {
		if options.IncludeStatusObservedGeneration {
			// Set status.observedGeneration if we're asked to do so.
			if err := unstructured.SetNestedField(h.after.Object, h.after.GetGeneration(), "status", "observedGeneration"); err != nil {
				return err
			}

			// Restore the changes back to the original object.
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(h.after.Object, obj); err != nil {
				return err
			}
		}
	}

	// Calculate and store the top-level field changes (e.g. "metadata", "spec", "status") we have before/after.
	h.changes, err = h.calculateChanges(obj)
	if err != nil {
		return err
	}

	return kerrors.NewAggregate([]error{
		// Then proceed to patch the rest of the object.
		h.patch(ctx, obj),
		h.patchStatus(ctx, obj),
	})
}

// patch issues a patch for metadata and spec.
func (h *Helper) patch(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch("metadata") && !h.shouldPatch("spec") {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj, specPatch)
	if err != nil {
		return err
	}
	return h.client.Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// patchStatus issues a patch if the status has changed.
func (h *Helper) patchStatus(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch("status") {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj, statusPatch)
	if err != nil {
		return err
	}
	return h.client.Status().Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// calculate changes tries to build a patch from the before/after objects we have
// and store in a map which top-level fields (e.g. `metadata`, `spec`, `status`, etc.) have changed.
func (h *Helper) calculateChanges(after client.Object) (map[string]bool, error) {
	// Calculate patch data.
	patch := client.MergeFrom(h.beforeObject)
	diff, err := patch.Data(after)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate patch data")
	}

	// Unmarshal patch data into a local map.
	patchDiff := map[string]interface{}{}
	if err := json.Unmarshal(diff, &patchDiff); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal patch data into a map")
	}

	// Return the map.
	res := make(map[string]bool, len(patchDiff))
	for key := range patchDiff {
		res[key] = true
	}
	return res, nil
}

// calculatePatch returns the before/after objects to be given in a controller-runtime patch, scoped down to the absolute necessary.
func (h *Helper) calculatePatch(afterObj client.Object, focus patchType) (client.Object, client.Object, error) {
	// Get a shallow unsafe copy of the before/after object in unstructured form.
	before := unsafeUnstructuredCopy(h.before, focus)
	after := unsafeUnstructuredCopy(h.after, focus)

	// We've now applied all modifications to local unstructured objects,
	// make copies of the original objects and convert them back.
	beforeObj := h.beforeObject.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(before.Object, beforeObj); err != nil {
		return nil, nil, err
	}
	afterObj = afterObj.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(after.Object, afterObj); err != nil {
		return nil, nil, err
	}
	return beforeObj, afterObj, nil
}

func (h *Helper) shouldPatch(in string) bool {
	return h.changes[in]
}

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	// If the incoming object is already unstructured, perform a deep copy first
	// otherwise DefaultUnstructuredConverter ends up returning the inner map without
	// making a copy.
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject()
	}
	rawMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: rawMap}, nil
}

func unsafeUnstructuredCopy(obj *unstructured.Unstructured, focus patchType) *unstructured.Unstructured {
	res := &unstructured.Unstructured{Object: make(map[string]interface{}, len(obj.Object))}
	for key := range obj.Object {
		value := obj.Object[key]

		// Perform a shallow copy only for the keys we're interested in, or the ones that should be always preserved.
		if key == focus.Key() || preserveUnstructuredKeys[key] {
			res.Object[key] = value
		}
	}

	return res
}

func unstructuredHasStatus(u *unstructured.Unstructured) bool {
	_, ok := u.Object["status"]
	return ok
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice, reflect.Interface, reflect.UnsafePointer, reflect.Func:
		return reflect.ValueOf(i).IsValid() && reflect.ValueOf(i).IsNil()
	}
	return false
}
