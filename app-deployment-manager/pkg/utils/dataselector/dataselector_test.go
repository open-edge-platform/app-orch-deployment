// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"reflect"
	"testing"
)

const (
	testFieldName1 = "name"
	testFieldName2 = "id"
	testFieldName3 = "description"
)

type PaginationTestCase struct {
	Info            string
	PaginationQuery *PaginationQuery
	ExpectedOrder   []int
}

type SortTestCase struct {
	Info          string
	SortQuery     *SortQuery
	ExpectedOrder []int
}

type FilterTestCase struct {
	Info             string
	FilterQuery      *FilterQuery
	ExpectedDataItem []TestDataItem
}

type TestDataItem struct {
	Name        string
	Id          int
	Description string
}

func (dc TestDataItem) GetField(name FieldName) Comparable {
	switch name {
	case testFieldName1:
		return StdComparableString(dc.Name)
	case testFieldName2:
		return StdComparableInt(dc.Id)
	case testFieldName3:
		return StdComparableString(dc.Description)

	default:
		return nil
	}
}

func toDataItem(std []TestDataItem) []DataItem {
	dataItems := make([]DataItem, len(std))
	for i := range std {
		dataItems[i] = std[i]
	}
	return dataItems
}

func fromDataItems(dataItems []DataItem) []TestDataItem {
	testDataItems := make([]TestDataItem, len(dataItems))
	for i := range testDataItems {
		testDataItems[i] = dataItems[i].(TestDataItem)
	}
	return testDataItems
}

func getDataItemList() []DataItem {
	return toDataItem([]TestDataItem{
		{"ab", 1, "ab-1"},
		{"ab", 2, "ab-2"},
		{"ab", 3, "ab-3"},
		{"ac", 4, "ac-4"},
		{"ac", 5, "ac-5"},
		{"ad", 6, "ad-6"},
		{"ba", 7, "ba-7"},
		{"da", 8, "da-8"},
		{"ea", 9, "ea-9"},
		{"aa", 10, "aa-10"},
	})
}

func getOrder(dataList []TestDataItem) []int {
	idOrder := []int{}
	for _, e := range dataList {
		idOrder = append(idOrder, e.Id)
	}
	return idOrder
}

func TestSortOrderBy(t *testing.T) {
	testCases := []SortTestCase{
		{
			"no sort - do not change the original order",
			NoSort,
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"ascending sort by 1 property - all items sorted by this property",
			NewSortQueryOrderBy([]OrderBy{
				{
					Field:     "id",
					Ascending: true,
				},
			}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"descending sort by 1 property - all items sorted by this property",
			NewSortQueryOrderBy([]OrderBy{
				{
					Field:     "id",
					Ascending: false,
				},
			}),
			[]int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
		{
			"sort by 2 properties - items should first be sorted by first property and later by second",
			NewSortQueryOrderBy([]OrderBy{
				{
					Field:     "name",
					Ascending: true,
				},
				{
					Field:     "id",
					Ascending: false,
				},
			}),
			[]int{10, 3, 2, 1, 5, 4, 6, 7, 8, 9},
		},
		{
			"empty sort list - no sort",
			NewSortQueryOrderBy([]OrderBy{}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"nil - no sort",
			NewSortQueryOrderBy(nil),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		// Invalid arguments to the NewSortQuery
		{
			"sort by few properties where at least one property name is invalid - no sort",
			NewSortQueryOrderBy([]OrderBy{
				{
					Field:     "INVALID_FIELD",
					Ascending: true,
				},
				{
					Field:     "id",
					Ascending: false,
				},
			}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"sort by few properties where one order tag is missing property - no sort",
			NewSortQueryOrderBy([]OrderBy{{}}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}
	for _, testCase := range testCases {
		selectableData := DataSelector{
			GenericDataList: getDataItemList(),
			DataSelectQuery: &DataSelectQuery{SortQuery: testCase.SortQuery},
		}
		sortedData := fromDataItems(selectableData.Sort().GenericDataList)
		order := getOrder(sortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Sort: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.SortQuery, order, testCase.ExpectedOrder)
		}
	}

}

func TestSort(t *testing.T) {
	testCases := []SortTestCase{
		{
			"no sort - do not change the original order",
			NoSort,
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"ascending sort by 1 property - all items sorted by this property",
			NewSortQuery([]string{ASC, "id"}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"descending sort by 1 property - all items sorted by this property",
			NewSortQuery([]string{DECS, "id"}),
			[]int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
		{
			"sort by 2 properties - items should first be sorted by first property and later by second",
			NewSortQuery([]string{ASC, "name", DECS, "id"}),
			[]int{10, 3, 2, 1, 5, 4, 6, 7, 8, 9},
		},
		{
			"empty sort list - no sort",
			NewSortQuery([]string{}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"nil - no sort",
			NewSortQuery(nil),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		// Invalid arguments to the NewSortQuery
		{
			"sort by few properties where at least one property name is invalid - no sort",
			NewSortQuery([]string{ASC, "INVALID_PROPERTY", DECS, "id"}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"sort by few properties where at least one order option is invalid - no sort",
			NewSortQuery([]string{DECS, "name", "INVALID_ORDER", "id"}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"sort by few properties where one order tag is missing property - no sort",
			NewSortQuery([]string{""}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"sort by few properties where one order tag is missing property - no sort",
			NewSortQuery([]string{DECS, "name", ASC, "id", ASC}),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}
	for _, testCase := range testCases {
		selectableData := DataSelector{
			GenericDataList: getDataItemList(),
			DataSelectQuery: &DataSelectQuery{SortQuery: testCase.SortQuery},
		}
		sortedData := fromDataItems(selectableData.Sort().GenericDataList)
		order := getOrder(sortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Sort: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.SortQuery, order, testCase.ExpectedOrder)
		}
	}
}

func TestPagination(t *testing.T) {
	testCases := []PaginationTestCase{
		{
			"empty pagination - no elements should be returned",
			EmptyPagination,
			[]int{},
		},
		{
			"request one item from existing page - element should be returned",
			NewPaginationQuery(1, 5),
			[]int{6},
		},
		{
			"request one item from non existing page - no elements should be returned",
			NewPaginationQuery(1, 10),
			[]int{},
		},
		{
			"request 2 items from existing page - 2 elements should be returned",
			NewPaginationQuery(2, 2),
			[]int{3, 4},
		},
		{
			"request 2 items from existing page - 2 elements should be returned",
			NewPaginationQuery(4, 8),
			[]int{9, 10},
		},
		{
			"request 3 items from partially existing page - last few existing should be returned",
			NewPaginationQuery(3, 9),
			[]int{10},
		},
		{
			"request more than total number of elements from page 1 - all existing elements should be returned",
			NewPaginationQuery(11, 0),
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"request 3 items from non existing page - no elements should be returned",
			NewPaginationQuery(3, 12),
			[]int{},
		},
	}
	for _, testCase := range testCases {
		selectableData := DataSelector{
			GenericDataList: getDataItemList(),
			DataSelectQuery: &DataSelectQuery{PaginationQuery: testCase.PaginationQuery},
		}
		paginatedData := fromDataItems(selectableData.Paginate().GenericDataList)
		order := getOrder(paginatedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}

}

func TestFilter(t *testing.T) {
	testCases := []FilterTestCase{
		{
			Info: "Filter by name",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("ab"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
			},
		},
		{
			Info: "Filter by name (upper case/lower case)",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("Ab"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
			},
		},
		{
			Info: "Filter by multiple names",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("ab"),
				},
				{
					Field: "name",
					Value: StdComparableString("da"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
				{
					Name:        "da",
					Id:          8,
					Description: "da-8",
				},
			},
		},
		{
			Info: "Filter by multiple names (upper case/lower case)",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("Ab"),
				},
				{
					Field: "name",
					Value: StdComparableString("dA"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
				{
					Name:        "da",
					Id:          8,
					Description: "da-8",
				},
			},
		},
		{
			Info: "Filter by name- wildcard at the end",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("ab*"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
			},
		},
		{
			Info: "wildcard at the the beginning",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("*b"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
			},
		},
		{
			Info: "wildcard at the end",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("b*"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ba",
					Id:          7,
					Description: "ba-7",
				},
			},
		},
		{
			Info: "Contains operations",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("a"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{"ab", 1, "ab-1"},
				{"ab", 2, "ab-2"},
				{"ab", 3, "ab-3"},
				{"ac", 4, "ac-4"},
				{"ac", 5, "ac-5"},
				{"ad", 6, "ad-6"},
				{"ba", 7, "ba-7"},
				{"da", 8, "da-8"},
				{"ea", 9, "ea-9"},
				{"aa", 10, "aa-10"},
			},
		},
		{
			Info: "Filter by multiple names",
			FilterQuery: NewFilterQueryFilterBy([]FilterBy{
				{
					Field: "name",
					Value: StdComparableString("ab"),
				},
				{
					Field: "name",
					Value: StdComparableString("da"),
				},
			}),
			ExpectedDataItem: []TestDataItem{
				{
					Name:        "ab",
					Id:          1,
					Description: "ab-1",
				},
				{
					Name:        "ab",
					Id:          2,
					Description: "ab-2",
				},
				{
					Name:        "ab",
					Id:          3,
					Description: "ab-3",
				},
				{
					Name:        "da",
					Id:          8,
					Description: "da-8",
				},
			},
		},
	}

	for _, testCase := range testCases {
		selectableData := DataSelector{
			GenericDataList: getDataItemList(),
			DataSelectQuery: &DataSelectQuery{FilterQuery: testCase.FilterQuery},
		}
		filteredData := fromDataItems(selectableData.Filter().GenericDataList)
		if !reflect.DeepEqual(filteredData, testCase.ExpectedDataItem) {
			t.Errorf(`Filtering: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.FilterQuery, filteredData, testCase.ExpectedDataItem)
		}
	}

}
