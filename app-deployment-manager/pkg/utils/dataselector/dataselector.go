// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"sort"
)

type FieldName string

type DataItem interface {
	GetField(FieldName) Comparable
}

// Comparable hold any value that can be compared to its own kind.
type Comparable interface {
	// Compare compares self with other value. Returns 1 if other value is smaller, 0 if they are the same, -1 if other is larger.
	Compare(Comparable) int
	// Contains returns true if self value contains or is equal to other value, false otherwise.
	Contains(Comparable) bool
}

// DataSelector SelectableData contains all the required data to perform data selection.
// It implements sort.Interface so its sortable under sort.Sort
type DataSelector struct {
	// GenericDataList hold generic data cells that are being selected.
	GenericDataList []DataItem
	// DataSelectQuery holds instructions for data select.
	DataSelectQuery *DataSelectQuery
}

// Len returns the length of data inside SelectableData.
func (ds DataSelector) Len() int { return len(ds.GenericDataList) }

// Swap swaps 2 indices inside SelectableData.
func (ds DataSelector) Swap(i, j int) {
	ds.GenericDataList[i], ds.GenericDataList[j] = ds.GenericDataList[j], ds.GenericDataList[i]
}

// Less compares 2 indices inside SelectableData and returns true if first index is larger.
func (ds DataSelector) Less(i, j int) bool {
	for _, orderBy := range ds.DataSelectQuery.SortQuery.OrderByList {
		a := ds.GenericDataList[i].GetField(orderBy.Field)
		b := ds.GenericDataList[j].GetField(orderBy.Field)
		// ignore sort completely if property name not found
		if a == nil || b == nil {
			break
		}
		cmp := a.Compare(b)
		if cmp != 0 {
			// values different
			return (cmp == -1 && orderBy.Ascending) || (cmp == 1 && !orderBy.Ascending)
		}
	}
	return false
}

// Sort sorts the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (ds *DataSelector) Sort() *DataSelector {
	sort.Sort(*ds)
	return ds
}

// Filter the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (ds *DataSelector) Filter() *DataSelector {
	var filteredList []DataItem
	if len(ds.DataSelectQuery.FilterQuery.FilterByList) == 0 {
		return ds
	}

	for _, c := range ds.GenericDataList {
		for _, filterBy := range ds.DataSelectQuery.FilterQuery.FilterByList {
			v := c.GetField(filterBy.Field)
			if v == nil {
				continue
			}
			if v.Contains(filterBy.Value) {
				filteredList = append(filteredList, c)
				break
			}
		}
	}

	ds.GenericDataList = filteredList
	return ds
}

// Paginate paginates the data inside as instructed by DataSelectQuery and returns itself to allow method chaining.
func (ds *DataSelector) Paginate() *DataSelector {
	pQuery := ds.DataSelectQuery.PaginationQuery
	dataList := ds.GenericDataList
	startIndex, endIndex := pQuery.GetPaginationSettings(len(dataList))

	// Return no items if requested page does not exist
	if !pQuery.IsPageAvailable(len(ds.GenericDataList), startIndex) {
		ds.GenericDataList = []DataItem{}
		return ds
	}

	ds.GenericDataList = dataList[startIndex:endIndex]
	return ds
}
