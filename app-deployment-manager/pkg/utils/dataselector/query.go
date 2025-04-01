// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

const (
	ASC  = "asc"
	DECS = "desc"
)

// DataSelectQuery is options for GenericDataSelect which takes []GenericDataCell and returns selected data.
// Can be extended to include any kind of selection - for example filtering.
// Currently included only Pagination and Sort options.
type DataSelectQuery struct {
	PaginationQuery *PaginationQuery
	SortQuery       *SortQuery
	FilterQuery     *FilterQuery
}

// SortQuery holds options for sort functionality of data select.
type SortQuery struct {
	OrderByList []OrderBy
}

// OrderBy holds the name of the field that should be sorted and whether order should be ascending or descending.
type OrderBy struct {
	Field     FieldName
	Ascending bool
}

// NoSort is as option for no sort.
var NoSort = &SortQuery{
	OrderByList: []OrderBy{},
}

type FilterQuery struct {
	FilterByList []FilterBy
}

type FilterBy struct {
	Field FieldName
	Value Comparable
}

var NoFilter = &FilterQuery{
	FilterByList: []FilterBy{},
}

// DefaultDataSelect downloads first 10 items from page 1 with no sort and no metrics.
var DefaultDataSelect = NewDataSelectQuery(DefaultPagination, NoSort, NoFilter)

// NewDataSelectQuery creates DataSelectQuery object from simpler data select queries.
func NewDataSelectQuery(paginationQuery *PaginationQuery, sortQuery *SortQuery, filterQuery *FilterQuery) *DataSelectQuery {
	return &DataSelectQuery{
		PaginationQuery: paginationQuery,
		SortQuery:       sortQuery,
		FilterQuery:     filterQuery,
	}
}

// NewSortQueryOrderBy sort query based on orderBy list
func NewSortQueryOrderBy(orderByList []OrderBy) *SortQuery {
	return &SortQuery{
		OrderByList: orderByList,
	}
}

// NewSortQuery takes raw sort options list and returns SortQuery object. For example:
// ["a", "parameter1", "d", "parameter2"] - means that the data should be sorted by
// parameter1 (ascending) and later - for results that return equal under parameter 1 sort - by parameter2 (descending)
func NewSortQuery(sortByListRaw []string) *SortQuery {
	if sortByListRaw == nil || len(sortByListRaw)%2 == 1 {
		// Empty sort list or invalid (odd) length
		return NoSort
	}
	var orderByList []OrderBy
	for i := 0; i+1 < len(sortByListRaw); i += 2 {
		// parse order option
		var ascending bool
		orderOption := sortByListRaw[i]
		if orderOption == ASC {
			ascending = true
		} else if orderOption == DECS {
			ascending = false
		} else {
			//  Invalid order option. Only ascending (a), descending (d) options are supported
			return NoSort
		}

		// parse field name
		fieldName := sortByListRaw[i+1]
		sortBy := OrderBy{
			Field:     FieldName(fieldName),
			Ascending: ascending,
		}
		// Add to the sort options.
		orderByList = append(orderByList, sortBy)
	}
	return &SortQuery{
		OrderByList: orderByList,
	}
}

// NewFilterQueryFilterBy creates a filter query based on filter by parameters list
func NewFilterQueryFilterBy(filterByList []FilterBy) *FilterQuery {
	return &FilterQuery{
		FilterByList: filterByList,
	}
}
