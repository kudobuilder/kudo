package helpers

import (
	"fmt"
	"github.com/google/go-github/github" // with go modules disabled
	"sort"
	"strconv"
)

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
