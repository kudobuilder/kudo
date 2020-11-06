package kudo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ SortableOperator = &sortableOp{}

type sortableOp struct {
	name       string
	ovVersion  string
	appVersion string
}

func (s sortableOp) OperatorName() string {
	return s.name
}

func (s sortableOp) OperatorVersion() string {
	return s.ovVersion
}

func (s sortableOp) AppVersion() string {
	return s.appVersion
}

func TestVersions(t *testing.T) {

	l := SortableOperatorList{
		sortableOp{name: "abc", appVersion: "aaa", ovVersion: "0.0.1"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "0.1.0"},
		sortableOp{name: "abc", appVersion: "aaa", ovVersion: "0.0.2"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "0.0.1"},
		sortableOp{name: "abc", appVersion: "0.0.1", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "1.1.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "10.1.0"},
		sortableOp{name: "abc", appVersion: "0.0.1", ovVersion: "10.1.0"},
		sortableOp{name: "abc", appVersion: "0.0.9", ovVersion: "1.0.1"},
		sortableOp{name: "abc", appVersion: "0.0.2", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "0.0.9", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "0.1.9", ovVersion: "1.0.1"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "1.0.2"},
		sortableOp{name: "abc", appVersion: "10.1.0", ovVersion: "0.1.0"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "0.0.1"},
		sortableOp{name: "abc", appVersion: "10.0.0", ovVersion: "0.1.1"},
		sortableOp{name: "abc", appVersion: "10.0.0", ovVersion: "0.1.0"},
		sortableOp{name: "cde", appVersion: "1.0.0", ovVersion: "1.0.0"},
	}

	sortedList := SortableOperatorList{
		sortableOp{name: "abc", appVersion: "10.1.0", ovVersion: "0.1.0"},
		sortableOp{name: "abc", appVersion: "10.0.0", ovVersion: "0.1.1"},
		sortableOp{name: "abc", appVersion: "10.0.0", ovVersion: "0.1.0"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "1.0.2"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "1.0.0", ovVersion: "0.0.1"},
		sortableOp{name: "abc", appVersion: "0.1.9", ovVersion: "1.0.1"},
		sortableOp{name: "abc", appVersion: "0.0.9", ovVersion: "1.0.1"},
		sortableOp{name: "abc", appVersion: "0.0.9", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "0.0.2", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "0.0.1", ovVersion: "10.1.0"},
		sortableOp{name: "abc", appVersion: "0.0.1", ovVersion: "1.0.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "10.1.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "1.1.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "0.1.0"},
		sortableOp{name: "abc", appVersion: "bbb", ovVersion: "0.0.1"},
		sortableOp{name: "abc", appVersion: "aaa", ovVersion: "0.0.2"},
		sortableOp{name: "abc", appVersion: "aaa", ovVersion: "0.0.1"},
		sortableOp{name: "cde", appVersion: "1.0.0", ovVersion: "1.0.0"},
	}

	filteredList := SortableOperatorList{
		sortableOp{name: "cde", appVersion: "1.0.0", ovVersion: "1.0.0"},
	}

	l.Sort()

	assert.Equal(t, l, sortedList)

	filtered := l.FilterByName("cde")
	assert.Equal(t, filtered, filteredList)
}
