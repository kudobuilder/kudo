package kudo

import (
	"sort"

	"github.com/Masterminds/semver"
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
	return funk.Filter(b, func(o SortableOperator) bool { return o.OperatorName() == name }).(SortableOperatorList)
}

func (b SortableOperatorList) Sort() {
	// We want to pick the newest (highest) versions first, that's why we reverse the order
	sort.Sort(sort.Reverse(b))
}

func (b SortableOperatorList) FindFirstMatch(name, operatorVersion, appVersion string) SortableOperator {
	for _, o := range b {
		ovFromList := o
		if name == ovFromList.OperatorName() &&
			(operatorVersion == "" || operatorVersion == ovFromList.OperatorVersion()) &&
			(appVersion == "" || appVersion == ovFromList.AppVersion()) {
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
		return b[x].OperatorName() < b[y].OperatorName()
	}

	// Compare OperatorVersion - Use SemVer if possible
	xVersion, _ := semver.NewVersion(b[x].OperatorVersion())
	yVersion, _ := semver.NewVersion(b[y].OperatorVersion())
	if xVersion != nil && yVersion != nil {
		res := xVersion.Compare(yVersion)
		if res != 0 {
			return res < 0
		}
	} else {
		if b[x].OperatorVersion() != b[y].OperatorVersion() {
			return b[x].OperatorVersion() < b[y].OperatorVersion()
		}
	}

	// Compare AppVersion - Use SemVer if possible
	xAppVersion, _ := semver.NewVersion(b[x].AppVersion())
	yAppVersion, _ := semver.NewVersion(b[y].AppVersion())
	if xAppVersion != nil && yAppVersion != nil {
		res := xAppVersion.Compare(yAppVersion)
		return res < 0
	}
	return b[x].AppVersion() < b[y].AppVersion()
}
