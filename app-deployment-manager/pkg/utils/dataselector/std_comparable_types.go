// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dataselector

import (
	"regexp"
	"strings"
	"time"
)

type StdComparableInt int

func (sc StdComparableInt) Compare(otherV Comparable) int {
	other := otherV.(StdComparableInt)
	return intsCompare(int(sc), int(other))
}

func (sc StdComparableInt) Contains(otherV Comparable) bool {
	return sc.Compare(otherV) == 0
}

type StdComparableString string

func (sc StdComparableString) Compare(otherV Comparable) int {
	other := otherV.(StdComparableString)
	lowerSc := strings.ToLower(string(sc))
	lowerOther := strings.ToLower(string(other))
	return strings.Compare(lowerSc, lowerOther)
}

func (sc StdComparableString) Contains(otherV Comparable) bool {
	other, ok := otherV.(StdComparableString)
	if !ok {
		return false
	}
	lowerOther := strings.ToLower(string(other))
	lowerSc := strings.ToLower(strings.ToLower(string(sc)))

	if strings.Contains(lowerOther, "*") {
		regexPattern := checkWildcardPattern(lowerOther)
		return matchesRegx(lowerSc, regexPattern)
	}

	return strings.Contains(lowerSc, lowerOther)
}

func checkWildcardPattern(pattern string) string {
	if strings.HasPrefix(pattern, "*") {
		pattern = ".*" + strings.TrimPrefix(pattern, "*")
	}
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	return "^" + pattern + "$"
}

func matchesRegx(fieldValue, pattern string) bool {
	match, err := regexp.MatchString(pattern, fieldValue)
	if err != nil {
		return false
	}
	return match
}

// StdComparableRFC3339Timestamp takes RFC3339 Timestamp strings and compares them as TIMES. In case of time parsing error compares values as strings.
type StdComparableRFC3339Timestamp string

func (sc StdComparableRFC3339Timestamp) Compare(otherV Comparable) int {
	other := otherV.(StdComparableRFC3339Timestamp)
	// try to compare as timestamp (earlier = smaller)
	selfTime, err1 := time.Parse(time.RFC3339, string(sc))
	otherTime, err2 := time.Parse(time.RFC3339, string(other))

	if err1 != nil || err2 != nil {
		// in case of timestamp parsing failure just compare as strings
		return strings.Compare(string(sc), string(other))
	}
	return ints64Compare(selfTime.Unix(), otherTime.Unix())
}

func (sc StdComparableRFC3339Timestamp) Contains(otherV Comparable) bool {
	return sc.Compare(otherV) == 0
}

type StdComparableTime time.Time

func (sc StdComparableTime) Compare(otherV Comparable) int {
	other := otherV.(StdComparableTime)
	return ints64Compare(time.Time(sc).Unix(), time.Time(other).Unix())
}

func (sc StdComparableTime) Contains(otherV Comparable) bool {
	return sc.Compare(otherV) == 0
}

// Int comparison functions. Similar to strings.Compare.
func intsCompare(a, b int) int {
	if a > b {
		return 1
	} else if a == b {
		return 0
	}
	return -1
}

func ints64Compare(a, b int64) int {
	if a > b {
		return 1
	} else if a == b {
		return 0
	}
	return -1
}
