// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"fmt"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 500
	MinPageSize     = 0
	MinOffset       = 0
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

// IsValidPagination returns true if pagination has non negative parameters and pageSize is within limits
func (p *PaginationQuery) IsValidPagination() error {
	// Check if PageSize is less than the minimum
	if p.PageSize < MinPageSize {
		return errors.NewInvalid(fmt.Sprintf("validation error:\n - page_size: value must be greater than or equal to %d [uint32.gte]", MinPageSize))
	}

	// Check if PageSize exceeds the maximum
	if p.PageSize > MaxPageSize {
		return errors.NewInvalid(fmt.Sprintf("validation error:\n - page_size: value must be less than or equal to %d [uint32.lte]", MaxPageSize))
	}

	// Check if Offset is less than the minimum
	if p.OffSet < MinOffset {
		return errors.NewInvalid("validation error:\n - offset: value must be greater than or equal to 0 [uint32.gte]")
	}

	return nil
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
