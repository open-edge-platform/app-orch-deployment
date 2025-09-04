// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
)

const (
	DefaultPageSize = 20 // Aligned with app-orch-catalog
	// MaxPageSize defines the maximum allowed page size to prevent excessive resource usage
	// Aligned with app-orch-catalog limit of 500 for consistency across platform
	MaxPageSize = 500
	// MinPageSize defines the minimum allowed page size
	MinPageSize = 0
	// MinOffset defines the minimum allowed offset value
	MinOffset = 0
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
	if p.PageSize < MinPageSize {
		return errors.NewInvalid("validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 500 [uint32.gte_lte]")
	}
	if p.PageSize > MaxPageSize {
		return errors.NewInvalid("validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to 500 [uint32.gte_lte]")
	}
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
