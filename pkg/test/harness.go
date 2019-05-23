package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
	"time"
)

// A TestCase contains the name of the test case, its index in the test,
// and all of the test case's settings (including objects to apply and assert on).
type TestCase struct {
	Name  string
	Index int

	Case   *kudo.TestCase
	Assert *kudo.TestAssert

	Asserts []runtime.Object
	Apply   []runtime.Object
	Errors  []runtime.Object

	Client client.Client
}

// Delete all resources defined in the Apply list.
func (t *TestCase) Clean(namespace string) error {
	for _, obj := range t.Apply {
		Namespaced(obj, namespace)

		log.Println("Deleting resource:", ResourceID(obj))
		if err := t.Client.Delete(context.TODO(), obj); err != nil {
			return err
		}
	}

	return nil
}

// Run a KUDO test case:
// 1. Apply all desired objects to Kubernetes.
// 2. Wait for all of the states defined in the test case's asserts to be true.'
func (t *TestCase) Run(namespace string) []error {
	log.Println("Running test case:", t.Name)

	testErrors := []error{}

	for _, obj := range t.Apply {
		_, _, err := Namespaced(obj, namespace)
		if err != nil {
			testErrors = append(testErrors, err)
			continue
		}

		log.Println("Creating resource:", ResourceID(obj))
		err = t.Client.Create(context.TODO(), obj)
		if err != nil && k8serrors.IsAlreadyExists(err) {
			log.Println("Resource already exists, updating:", ResourceID(obj))
			err = t.Client.Update(context.TODO(), obj)
		}

		if err != nil {
			testErrors = append(testErrors, err)
		}
	}

	if len(testErrors) != 0 {
		return testErrors
	}

	timeout := 10
	if t.Assert != nil && t.Assert.Timeout != 0 {
		timeout = t.Assert.Timeout
	}

	for i := 0; i < timeout; i++ {
		testErrors = []error{}

		for _, expected := range t.Asserts {
			name, namespace, err := Namespaced(expected, namespace)
			if err != nil {
				testErrors = append(testErrors, err)
				continue
			}

			gvk := expected.GetObjectKind().GroupVersionKind()

			actual := &unstructured.Unstructured{}
			actual.SetGroupVersionKind(gvk)

			err = t.Client.Get(context.TODO(), client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			}, actual)
			if err != nil {
				testErrors = append(testErrors, err)
				continue
			}

			expectedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(expected)
			if err != nil {
				testErrors = append(testErrors, err)
				continue
			}

			if err := IsSubset(expectedObj, actual.UnstructuredContent()); err != nil {
				testErrors = append(testErrors, fmt.Errorf("Error: resource %s: %s\n", ResourceID(expected), err))

				diff, err := PrettyDiff(expected, actual)
				if err == nil {
					testErrors = append(testErrors, errors.New(diff))
				} else {
					testErrors = append(testErrors, err)
				}
			}
		}

		if len(testErrors) == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	return testErrors
}

// Contains all of the test cases and the Kubernetes client and other global configuration
// for a test.
type Test struct {
	Cases  []*TestCase
	Name   string
	Dir    string
	Client client.Client
}

// Delete a namespace in Kubernetes after we are done using it.
func (t *Test) DeleteNamespace(namespace string) error {
	return t.Client.Delete(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	})
}

// Create a namespace in Kubernetes to use for a test.
func (t *Test) CreateNamespace(namespace string) error {
	return t.Client.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
	})
}

// Create a new Go test that runs a set of test cases.
func (t *Test) TestFactory() func(*testing.T) {
	return func(test *testing.T) {
		test.Parallel()

		ns, err := uuid.NewRandom()
		if err != nil {
			test.Fatal(err)
		}

		if err := t.CreateNamespace(ns.String()); err != nil {
			test.Fatal(err)
		}

		defer t.DeleteNamespace(ns.String())

		for _, testCase := range t.Cases {
			testCase.Client = t.Client

			defer testCase.Clean(ns.String())

			errs := testCase.Run(ns.String())

			for _, err := range errs {
				test.Error(err)
			}

			if len(errs) != 0 {
				break
			}
		}
	}
}

// If called from within a Go test (t), it will launch all of the KUDO integration tests at dir.
func RunHarness(dir string, t *testing.T) {
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		t.Fatal(err)
	}

	tests, err := LoadTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		test.Client = cl

		if err := test.LoadTestCases(); err != nil {
			t.Fatal(err)
		}

		t.Run(test.Name, test.TestFactory())
	}
}
