// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNewPaginationQuery(t *testing.T) {
	cases := []struct {
		itemsPerPage, page int
		expected           *PaginationQuery
	}{
		{0, 0, &PaginationQuery{0, 0}},
		{1, 10, &PaginationQuery{1, 10}},
	}

	for _, c := range cases {
		actual := NewPaginationQuery(c.itemsPerPage, c.page)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("NewPaginationQuery(%+v, %+v) == %+v, expected %+v",
				c.itemsPerPage, c.page, actual, c.expected)
		}
	}
}

func TestIsValidPagination(t *testing.T) {
	offsetErr := errors.NewInvalid("validation error:\n - offset: value must be greater than or equal to 0 [uint32.gte]")
	pageSizeErr := errors.NewInvalid("validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 100 [uint32.gte_lte]")
	pageSizeMaxErr := errors.NewInvalid("validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 100 [uint32.gte_lte]")
	cases := []struct {
		pQuery   *PaginationQuery
		expected error
	}{
		{&PaginationQuery{0, 0}, nil},
		{&PaginationQuery{5, 0}, nil},
		{&PaginationQuery{10, 1}, nil},
		{&PaginationQuery{0, 2}, nil},
		{&PaginationQuery{100, 0}, nil},
		{&PaginationQuery{101, 0}, pageSizeMaxErr},
		{&PaginationQuery{10, -1}, offsetErr},
		{&PaginationQuery{-1, 0}, pageSizeErr},
		{&PaginationQuery{-1, -1}, pageSizeErr}, // pagesize is checked first
	}

	for _, c := range cases {
		actual := c.pQuery.IsValidPagination()
		assert.Equal(t, actual, c.expected)
	}
}

func TestGetPaginationSettings(t *testing.T) {
	cases := []struct {
		pQuery               *PaginationQuery
		itemsCount           int
		startIndex, endIndex int
	}{
		{&PaginationQuery{0, 0}, 10, 0, 0},
		{&PaginationQuery{10, 10}, 10, 10, 10},
		{&PaginationQuery{10, 0}, 10, 0, 10},
	}

	for _, c := range cases {
		actualStartIdx, actualEndIdx := c.pQuery.GetPaginationSettings(c.itemsCount)
		if actualStartIdx != c.startIndex || actualEndIdx != c.endIndex {
			t.Errorf("GetPaginationSettings(%+v) == %+v, %+v, expected %+v, %+v",
				c.itemsCount, actualStartIdx, actualEndIdx, c.startIndex, c.endIndex)
		}
	}
}
