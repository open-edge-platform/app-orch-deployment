// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type OrderByTestCase struct {
	orderByParam        string
	expectedOrderByList []dataselector.OrderBy
}

type FilterTestCase struct {
	name             string
	filter           string
	expectedFilterBy []dataselector.FilterBy
	wantErr          bool
}

func TestParseOrderBy(t *testing.T) {
	testCases := []OrderByTestCase{
		{
			orderByParam: "name, id desc",
			expectedOrderByList: []dataselector.OrderBy{
				{
					Field:     dataselector.FieldName("name"),
					Ascending: true,
				},
				{
					Field:     dataselector.FieldName("id"),
					Ascending: false,
				},
			},
		},
		{
			orderByParam: "name asc, id asc",
			expectedOrderByList: []dataselector.OrderBy{
				{
					Field:     dataselector.FieldName("name"),
					Ascending: true,
				},
				{
					Field:     dataselector.FieldName("id"),
					Ascending: true,
				},
			},
		},
	}
	for _, testCase := range testCases {
		orderByList, err := ParseOrderBy(testCase.orderByParam)
		if err == nil {
			assert.Equal(t, testCase.expectedOrderByList, orderByList)
		}
	}

}

func TestParseFilterBy(t *testing.T) {
	tests := []FilterTestCase{
		{
			name:   "Single filter test case",
			filter: "name=abc",
			expectedFilterBy: []dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("abc"),
				},
			},
		},
		{
			name:   "Multiple filters with OR",
			filter: "name=abc OR age=30",
			expectedFilterBy: []dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("abc"),
				},
				{
					Field: "age",
					Value: dataselector.StdComparableString("30"),
				},
			},
		},
		{
			name:   "Wildcard at the end",
			filter: "name=abc*",
			expectedFilterBy: []dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("abc*"),
				},
			},
		},
		{
			name:   "Wildcard at the beginning",
			filter: "description=*xyz",
			expectedFilterBy: []dataselector.FilterBy{
				{
					Field: "description",
					Value: dataselector.StdComparableString("*xyz"),
				},
			},
		},
		{
			name:   "multiple name filtering",
			filter: "name=abc OR name=def",
			expectedFilterBy: []dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("abc"),
				},
				{
					Field: "name",
					Value: dataselector.StdComparableString("def"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := ParseFilterBy(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilterBy error = %v, exptedErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(output, tt.expectedFilterBy) {
				t.Errorf("ParseFilterBy got %v, expected %v", output, tt.expectedFilterBy)
			}

		})
	}

}
