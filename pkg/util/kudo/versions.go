package kudo

import (
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/thoas/go-funk"
)

type SortableOperator interface {
	OperatorName() string
	OperatorVersion() string
	AppVersion() string
}

type SortableOperatorList []SortableOperator

func (b SortableOperatorList) FilterByName(name string) SortableOperatorList {
	// nolint:errcheck
	return funk.Filter(b, func(o SortableOperator) bool { return o.OperatorName() == name }).([]SortableOperator)
}

func (b SortableOperatorList) Sort() {
	// We want to pick the newest (highest) versions first, that's why we reverse the order
	sort.Sort(sort.Reverse(b))
}

func (b SortableOperatorList) FindFirstMatch(name, operatorVersion, appVersion string) SortableOperator {
	for _, o := range b {
		o := o
		if name == o.OperatorName() &&
			(operatorVersion == "" || operatorVersion == o.OperatorVersion()) &&
			(appVersion == "" || appVersion == o.AppVersion()) {
			return o
		}
	}
	return nil
}

// Len returns the number of entries
// This is needed to allow sorting.
func (b SortableOperatorList) Len() int { return len(b) }

// Swap swaps the position of two items in the slice.
// This is needed to allow sorting.
func (b SortableOperatorList) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// This is needed to allow sorting.
func (b SortableOperatorList) Less(x, y int) bool {
	// First compare Operator name
	if b[x].OperatorName() != b[y].OperatorName() {
		// We compare in the other direction here - this way we get the operator names sorted alphabetically,
		// and the version from high to low
		return b[x].OperatorName() > b[y].OperatorName()
	}

	avCompare := compareVersion(b[x].AppVersion(), b[y].AppVersion())
	if avCompare != 0 {
		return avCompare < 0
	}

	ovCompare := compareVersion(b[x].OperatorVersion(), b[y].OperatorVersion())
	return ovCompare < 0
}

// Compares two versions - tries to use semantic versioning first, falls back to string compare.
// non-semantic versions are always ordered lower than semantic ones
// abc
// cde
// 1.0.0
// 1.0.1
// 2.0.0
// 2.1.0
// 10.0.0
func compareVersion(x, y string) int {
	if x == y {
		return 0
	}
	xVersion, _ := semver.NewVersion(x)
	yVersion, _ := semver.NewVersion(y)

	if xVersion != nil && yVersion != nil {
		return xVersion.Compare(yVersion)
	}

	if xVersion == nil && yVersion == nil {
		if x < y {
			return -1
		}
		return 1
	}
	if xVersion == nil {
		return -1
	}
	if yVersion == nil {
		return 1
	}
	return 1
}
