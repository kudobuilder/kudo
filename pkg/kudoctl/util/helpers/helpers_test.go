package helpers

import (
	"reflect"
	"testing"

	"github.com/google/go-github/github"
)

func TestSortDirectoryContent(t *testing.T) {

	// For test case #1
	expectedEmptyRepositoryErrors := []string{
		"empty repository slice",
	}

	// For test case #2
	name0 := "0"
	name1 := "1"
	directoryContentSorted := []*github.RepositoryContent{
		{Name: &name0},
		{Name: &name1},
	}
	directoryContentUnsorted := []*github.RepositoryContent{
		{Name: &name1},
		{Name: &name0},
	}

	tests := []struct {
		in       []*github.RepositoryContent
		expected []*github.RepositoryContent
		err      []string
	}{
		{nil, nil, expectedEmptyRepositoryErrors},               // 1
		{directoryContentSorted, directoryContentSorted, nil},   // 2
		{directoryContentUnsorted, directoryContentSorted, nil}, // 3
	}

	for i, tt := range tests {
		i := i
		actual, err := SortDirectoryContent(tt.in)
		if err != nil {
			receivedErrorList := []string{err.Error()}
			diff := compareSlice(receivedErrorList, tt.err)
			for _, err := range diff {
				t.Errorf("%d: Found unexpected error: %v", i+1, err)
			}

			missing := compareSlice(tt.err, receivedErrorList)
			for _, err := range missing {
				t.Errorf("%d: Missed expected error: %v", i+1, err)
			}
		}

		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.expected, actual)
		}
	}
}

func TestPosString(t *testing.T) {

	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	response3 := "no"
	response4 := "y"

	tests := []struct {
		slice    []string
		element  string
		expected int
	}{
		{nil, "", -1},                  // 1
		{okayResponses, "", -1},        // 2
		{okayResponses, response3, -1}, // 3
		{okayResponses, response4, 0},  // 4
	}

	for i, tt := range tests {
		i := i
		actual := posString(tt.slice, tt.element)

		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.expected, actual)
		}
	}
}

func TestContainsString(t *testing.T) {

	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	response3 := "no"
	response4 := "y"

	tests := []struct {
		slice    []string
		element  string
		expected bool
	}{
		{nil, "", false},                  // 1
		{okayResponses, "", false},        // 2
		{okayResponses, response3, false}, // 3
		{okayResponses, response4, true},  // 4
	}

	for i, tt := range tests {
		i := i
		actual := containsString(tt.slice, tt.element)

		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("%d:\nexpected: %+v\n     got: %+v", i+1, tt.expected, actual)
		}
	}
}

func compareSlice(real, mock []string) []string {
	lm := len(mock)

	var diff []string

	for _, rv := range real {
		i := 0
		j := 0
		for _, mv := range mock {
			i++
			if rv == mv {
				continue
			}
			if rv != mv {
				j++
			}
			if lm <= j {
				diff = append(diff, rv)
			}
		}
	}
	return diff
}
