package test

import (
	"fmt"
	"sort"
	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"regexp"
	"strconv"
)

var testCaseRegex = regexp.MustCompile(`^(\d+)-([^.]+)(.yaml)?$`)
var fileNameRegex = regexp.MustCompile(`^(\d+-)?([^.]+)(.yaml)?$`)

// Load the resources from a YAML file for a test case:
// * If the YAML file is called "assert", then it contains objects to
//   add to the test case's list of assertions.
// * If the YAML file is called "errors", then it contains objects that,
//   if seen, mark a test immediately failed.
// * All other YAML files are considered resources to create.
func (tc *TestCase) LoadYAML(file string) error {
	objects, err := LoadYAML(file)
	if err != nil {
		return fmt.Errorf("loading %s: %s", file, err)
	}

	matches := fileNameRegex.FindStringSubmatch(filepath.Base(file))
	fname := matches[2]

	switch fname {
	case "assert":
		tc.Asserts = append(tc.Asserts, objects...)
	case "errors":
		tc.Errors = append(tc.Errors, objects...)
	default:
		if tc.Name == "" {
			tc.Name = fname
		}
		tc.Apply = append(tc.Apply, objects...)
	}

	asserts := []runtime.Object{}

	for _, obj := range tc.Asserts {
		if obj.GetObjectKind().GroupVersionKind().Kind == "TestAssert" {
			tc.Assert = obj.(*kudo.TestAssert)
		} else {
			asserts = append(asserts, obj)
		}
	}

	apply := []runtime.Object{}

	for _, obj := range tc.Apply {
		if obj.GetObjectKind().GroupVersionKind().Kind == "TestCase" {
			tc.Case = obj.(*kudo.TestCase)
			tc.Case.Index = tc.Index
			if tc.Case.Name != "" {
				tc.Name = tc.Case.Name
			}
		} else {
			apply = append(apply, obj)
		}
	}

	tc.Apply = apply
	tc.Asserts = asserts
	return nil
}

// Load all of the tests in a given directory.
func LoadTests(dir string) ([]*Test, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	tests := []*Test{}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		tests = append(tests, &Test{
			Cases: []*TestCase{},
			Name: file.Name(),
			Dir: filepath.Join(dir, file.Name()),
		})
	}

	return tests, nil
}

// Collect a map of test cases and their associated files from a directory.
func (test *Test) CollectTestCaseFiles() (map[int64][]string, error) {
	testCaseFiles := map[int64][]string{}

	files, err := ioutil.ReadDir(test.Dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		matches := testCaseRegex.FindStringSubmatch(file.Name())

		index, err := strconv.ParseInt(matches[1], 10, 32)
		if err != nil {
			return nil, err
		}

		if testCaseFiles[index] == nil {
			testCaseFiles[index] = []string{}
		}

		testCasePath := filepath.Join(test.Dir, file.Name())

		if file.IsDir() {
			testCaseDir, err := ioutil.ReadDir(testCasePath)
			if err != nil {
				return nil, err
			}

			for _, testCaseFile := range testCaseDir {
				testCaseFiles[index] = append(testCaseFiles[index], filepath.Join(
					testCasePath, testCaseFile.Name(),
				))
			}
		} else {
			testCaseFiles[index] = append(testCaseFiles[index], testCasePath)
		}
	}

	return testCaseFiles, nil
}

// Load all of the test cases for a test.
func (test *Test) LoadTestCases() error {
	testCaseFiles, err := test.CollectTestCaseFiles()
	if err != nil {
		return err
	}

	testCases := []*TestCase{}

	for index, files := range testCaseFiles {
		testCase := &TestCase{
			Index:   int(index),
			Asserts: []runtime.Object{},
			Apply:   []runtime.Object{},
			Errors:  []runtime.Object{},
		}

		for _, file := range files {
			if err := testCase.LoadYAML(file); err != nil {
				return err
			}
		}

		testCases = append(testCases, testCase)
	}

	sort.Slice(testCases, func(i, j int) bool {
		return testCases[i].Index < testCases[j].Index
	})

	test.Cases = testCases
	return nil
}
