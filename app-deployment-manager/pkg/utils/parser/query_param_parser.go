// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"regexp"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"strings"
)

func ParseOrderBy(orderByParameter string) ([]dataselector.OrderBy, error) {
	var orderBys []dataselector.OrderBy
	if orderByParameter == "" {
		return nil, nil
	}
	fields := strings.Split(orderByParameter, ",")

	for _, field := range fields {
		parts := strings.Fields(strings.TrimSpace(field))

		if len(parts) == 0 || len(parts) > 2 {
			return nil, errors.NewInvalid("invalid format for order by parameter")
		}

		asc := true
		if len(parts) == 2 {
			direction := strings.ToLower(parts[1])
			if direction == "desc" {
				asc = false
			} else if direction != "asc" {
				return nil, errors.NewInvalid("invalid order direction; must be 'asc' or 'desc'")
			}
		}

		orderBys = append(orderBys, dataselector.OrderBy{
			Field:     dataselector.FieldName(parts[0]),
			Ascending: asc,
		})

	}
	return orderBys, nil
}

func ParseFilterBy(filterByParameter string) ([]dataselector.FilterBy, error) {
	if filterByParameter == "" {
		return nil, nil
	}

	var filterBys []dataselector.FilterBy
	if strings.HasPrefix(filterByParameter, `"`) && strings.HasSuffix(filterByParameter, `"`) {
		// Handle when filterByParameter string equals single double
		// quote ("), this will prevent slice bounds out of range
		if len(filterByParameter) != 1 {
			filterByParameter = filterByParameter[1 : len(filterByParameter)-1]
		}
	}

	r := regexp.MustCompile(`([^=]+)[ \t]*=[ \t]*(.+)`)
	orExpressions := strings.Split(filterByParameter, " OR ")

	for _, expr := range orExpressions {
		matches := r.FindStringSubmatch(expr)
		if len(matches) < 3 {
			return nil, errors.NewInvalid("invalid filter request")
		}

		filterBy := dataselector.FilterBy{
			Field: dataselector.FieldName(matches[1]),
			Value: dataselector.StdComparableString(matches[2]),
		}
		filterBys = append(filterBys, filterBy)
	}

	return filterBys, nil
}
