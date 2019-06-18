package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testStepRegex = regexp.MustCompile(`^(\d+)-([^.]+)(.yaml)?$`)

// Case contains all of the test steps and the Kubernetes client and other global configuration
// for a test.
type Case struct {
	Steps []*Step
	Name  string
	Dir   string

	Client          client.Client
	DiscoveryClient discovery.DiscoveryInterface
	Logger          testutils.Logger
}

// DeleteNamespace deletes a namespace in Kubernetes after we are done using it.
func (t *Case) DeleteNamespace(namespace string) error {
	t.Logger.Log("Deleting namespace:", namespace)
	return t.Client.Delete(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	})
}

// CreateNamespace creates a namespace in Kubernetes to use for a test.
func (t *Case) CreateNamespace(namespace string) error {
	t.Logger.Log("Creating namespace:", namespace)
	return t.Client.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	})
}

// TestCaseFactory creates a new Go test that runs a set of test steps.
func (t *Case) TestCaseFactory() func(*testing.T) {
	return func(test *testing.T) {
		t.Logger = testutils.NewTestLogger(test, t.Name)

		test.Parallel()

		ns := fmt.Sprintf("kudo-test-%s", petname.Generate(2, "-"))

		if err := t.CreateNamespace(ns); err != nil {
			test.Fatal(err)
		}

		defer t.DeleteNamespace(ns)

		for _, testStep := range t.Steps {
			testStep.Client = t.Client
			testStep.DiscoveryClient = t.DiscoveryClient
			testStep.Logger = t.Logger.WithPrefix(testStep.String())

			defer testStep.Clean(ns)

			if errs := testStep.Run(ns); len(errs) > 0 {
				for _, err := range errs {
					test.Error(err)
				}

				test.Error(fmt.Errorf("failed in step %s", testStep.String()))
				break
			}
		}
	}
}

// CollectTestStepFiles collects a map of test steps and their associated files
// from a directory.
func (t *Case) CollectTestStepFiles() (map[int64][]string, error) {
	testStepFiles := map[int64][]string{}

	files, err := ioutil.ReadDir(t.Dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		matches := testStepRegex.FindStringSubmatch(file.Name())

		index, err := strconv.ParseInt(matches[1], 10, 32)
		if err != nil {
			return nil, err
		}

		if testStepFiles[index] == nil {
			testStepFiles[index] = []string{}
		}

		testStepPath := filepath.Join(t.Dir, file.Name())

		if file.IsDir() {
			testStepDir, err := ioutil.ReadDir(testStepPath)
			if err != nil {
				return nil, err
			}

			for _, testStepFile := range testStepDir {
				testStepFiles[index] = append(testStepFiles[index], filepath.Join(
					testStepPath, testStepFile.Name(),
				))
			}
		} else {
			testStepFiles[index] = append(testStepFiles[index], testStepPath)
		}
	}

	return testStepFiles, nil
}

// LoadTestSteps loads all of the test steps for a test case.
func (t *Case) LoadTestSteps() error {
	testStepFiles, err := t.CollectTestStepFiles()
	if err != nil {
		return err
	}

	testSteps := []*Step{}

	for index, files := range testStepFiles {
		testStep := &Step{
			Index:   int(index),
			Asserts: []runtime.Object{},
			Apply:   []runtime.Object{},
			Errors:  []runtime.Object{},
		}

		for _, file := range files {
			if err := testStep.LoadYAML(file); err != nil {
				return err
			}
		}

		testSteps = append(testSteps, testStep)
	}

	sort.Slice(testSteps, func(i, j int) bool {
		return testSteps[i].Index < testSteps[j].Index
	})

	t.Steps = testSteps
	return nil
}
