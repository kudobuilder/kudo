package helpers

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/google/go-github/github" // with go modules disabled
)

// AskForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling askForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)").
// Kindly borrowed from https://gist.github.com/albrow/5882501
func AskForConfirmation() bool {
	var response string
	fmt.Scanln(&response)

	if response == "" {
		return true
	}

	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Print("Please type Yes or no and then press enter: ")
		return AskForConfirmation()
	}
}

// SortDirectoryContent sorts a versions directory in descending order
func SortDirectoryContent(dc []*github.RepositoryContent) ([]*github.RepositoryContent, error) {
	if len(dc) < 1 {
		return nil, fmt.Errorf("empty repository slice")
	}
	sortedDirectoryContent := dc
	// Sorting with highest number first
	sort.Slice(sortedDirectoryContent, func(i, j int) bool {
		v1, _ := strconv.Atoi(*sortedDirectoryContent[i].Name)
		v2, _ := strconv.Atoi(*sortedDirectoryContent[j].Name)
		return v1 > v2
	})
	return sortedDirectoryContent, nil
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true if slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}
