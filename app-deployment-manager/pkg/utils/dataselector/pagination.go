// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

const (
	DefaultPageSize = 10
)

// EmptyPagination No items will be returned
var EmptyPagination = NewPaginationQuery(0, 0)

// DefaultPagination Returns 10 items from page 1
var DefaultPagination = NewPaginationQuery(10, 0)

// PaginationQuery structure represents pagination settings
type PaginationQuery struct {
	// How many items per page should be returned
	PageSize int
	// Number of page that should be returned when pagination is applied to the list
	// (PageNo - 1) * PageSize
	OffSet int
}

// NewPaginationQuery return pagination query structure based on given parameters
func NewPaginationQuery(pageSize, offSet int) *PaginationQuery {
	return &PaginationQuery{pageSize, offSet}
}

// IsValidPagination returns true if pagination has non negative parameters
func (p *PaginationQuery) IsValidPagination() error {
	if p.PageSize >= 0 && p.OffSet >= 0 {
		return nil
	}
	return errors.NewInvalid("pagesize and offset parameters must be gte zero")
}

// IsPageAvailable returns true if at least one element can be placed on page. False otherwise
func (p *PaginationQuery) IsPageAvailable(itemsCount, startingIndex int) bool {
	return itemsCount > startingIndex && p.PageSize > 0
}

// GetPaginationSettings based on number of items and pagination query parameters returns start
// and end index that can be used to return paginated list of items.
func (p *PaginationQuery) GetPaginationSettings(itemsCount int) (startIndex int, endIndex int) {
	startIndex = p.OffSet
	endIndex = startIndex + p.PageSize
	if endIndex < 0 {
		endIndex = startIndex
	}

	if endIndex > itemsCount {
		endIndex = itemsCount
	}

	return startIndex, endIndex
}
