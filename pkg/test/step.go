package test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var fileNameRegex = regexp.MustCompile(`^(\d+-)?([^.]+)(.yaml)?$`)

// A Step contains the name of the test step, its index in the test,
// and all of the test step's settings (including objects to apply and assert on).
type Step struct {
	Name  string
	Index int

	Step   *kudo.TestStep
	Assert *kudo.TestAssert

	Asserts []runtime.Object
	Apply   []runtime.Object
	Errors  []runtime.Object

	DiscoveryClient discovery.DiscoveryInterface
	Client          client.Client
	Logger          testutils.Logger
}

// Clean deletes all resources defined in the Apply list.
func (s *Step) Clean(namespace string) error {
	for _, obj := range s.Apply {
		_, _, err := testutils.Namespaced(s.DiscoveryClient, obj, namespace)
		if err != nil {
			return err
		}

		if err := s.Client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// DeleteExisting deletes any resources in the TestStep.Delete list prior to running the tests.
func (s *Step) DeleteExisting(namespace string) error {
	toDelete := []runtime.Object{}

	if s.Step == nil {
		return nil
	}

	for _, ref := range s.Step.Delete {
		gvk := ref.GroupVersionKind()

		obj := testutils.NewResource(gvk.GroupVersion().String(), gvk.Kind, ref.Name, "")

		objNs := namespace
		if ref.Namespace != "" {
			objNs = ref.Namespace
		}

		_, objNs, err := testutils.Namespaced(s.DiscoveryClient, obj, objNs)
		if err != nil {
			return err
		}

		if ref.Labels != nil && len(ref.Labels) != 0 {
			// If the reference has a label selector, List all objects that match
			if err := testutils.Retry(context.TODO(), func(ctx context.Context) error {
				u := &unstructured.UnstructuredList{}
				u.SetGroupVersionKind(gvk)

				listOptions := []client.ListOptionFunc{client.MatchingLabels(ref.Labels)}
				if objNs != "" {
					listOptions = append(listOptions, client.InNamespace(objNs))
				}

				err := s.Client.List(ctx, u, listOptions...)
				if err != nil {
					return errors.Wrap(err, "listing matching resources")
				}

				for index := range u.Items {
					toDelete = append(toDelete, &u.Items[index])
				}

				return nil
			}, testutils.IsJSONSyntaxError); err != nil {
				return err
			}
		} else {
			// Otherwise just append the object specified.
			toDelete = append(toDelete, obj.DeepCopyObject())
		}
	}

	for _, obj := range toDelete {
		if err := testutils.Retry(context.TODO(), func(ctx context.Context) error {
			err := s.Client.Delete(context.TODO(), obj.DeepCopyObject())
			if err != nil && k8serrors.IsNotFound(err) {
				return nil
			}
			return err
		}, testutils.IsJSONSyntaxError); err != nil {
			return err
		}
	}

	// Wait for resources to be deleted.
	return wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (done bool, err error) {
		for _, obj := range toDelete {
			err = s.Client.Get(context.TODO(), testutils.ObjectKey(obj), obj.DeepCopyObject())
			if err == nil || !k8serrors.IsNotFound(err) {
				return false, err
			}
		}

		return true, nil
	})
}

// Create applies all resources defined in the Apply list.
func (s *Step) Create(namespace string) []error {
	errors := []error{}

	for _, obj := range s.Apply {
		_, _, err := testutils.Namespaced(s.DiscoveryClient, obj, namespace)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if updated, err := testutils.CreateOrUpdate(context.TODO(), s.Client, obj, true); err != nil {
			errors = append(errors, err)
		} else {
			action := "created"
			if updated {
				action = "updated"
			}
			s.Logger.Log(testutils.ResourceID(obj), action)
		}
	}

	return errors
}

// GetTimeout gets the timeout defined for the test step.
func (s *Step) GetTimeout() int {
	timeout := 10
	if s.Assert != nil && s.Assert.Timeout != 0 {
		timeout = s.Assert.Timeout
	}
	return timeout
}

// CheckResource checks if the expected resource's state in Kubernetes is correct.
func (s *Step) CheckResource(expected runtime.Object, namespace string) []error {
	testErrors := []error{}

	name, namespace, err := testutils.Namespaced(s.DiscoveryClient, expected, namespace)
	if err != nil {
		return append(testErrors, err)
	}

	gvk := expected.GetObjectKind().GroupVersionKind()

	actuals := []*unstructured.Unstructured{}

	if name != "" {
		actual := &unstructured.Unstructured{}
		actual.SetGroupVersionKind(gvk)

		err = s.Client.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, actual)

		actuals = append(actuals, actual)
	} else {
		actual := &unstructured.UnstructuredList{}
		actual.SetGroupVersionKind(gvk)

		listOptions := []client.ListOptionFunc{}

		if namespace != "" {
			listOptions = append(listOptions, client.InNamespace(namespace))
		}

		err = s.Client.List(context.TODO(), actual, listOptions...)

		for _, item := range actual.Items {
			actuals = append(actuals, &item)
		}
	}
	if err != nil {
		return append(testErrors, err)
	}

	expectedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(expected)
	if err != nil {
		return append(testErrors, err)
	}

	for _, actual := range actuals {
		tmpTestErrors := []error{}

		if err := testutils.IsSubset(expectedObj, actual.UnstructuredContent()); err != nil {
			diff, diffErr := testutils.PrettyDiff(expected, actual)
			if diffErr == nil {
				tmpTestErrors = append(tmpTestErrors, errors.New(diff))
			} else {
				tmpTestErrors = append(tmpTestErrors, diffErr)
			}

			tmpTestErrors = append(tmpTestErrors, fmt.Errorf("resource %s: %s", testutils.ResourceID(expected), err))
		}

		if len(tmpTestErrors) == 0 {
			return tmpTestErrors
		}

		testErrors = append(testErrors, tmpTestErrors...)
	}

	return testErrors
}

// Check checks if the resources defined in Asserts are in the correct state.
func (s *Step) Check(namespace string) []error {
	testErrors := []error{}

	for _, expected := range s.Asserts {
		testErrors = append(testErrors, s.CheckResource(expected, namespace)...)
	}

	return testErrors
}

// Run runs a KUDO test step:
// 1. Apply all desired objects to Kubernetes.
// 2. Wait for all of the states defined in the test step's asserts to be true.'
func (s *Step) Run(namespace string) []error {
	s.Logger.Log("starting test step", s.String())

	if err := s.DeleteExisting(namespace); err != nil {
		return []error{err}
	}

	testErrors := s.Create(namespace)

	if len(testErrors) != 0 {
		return testErrors
	}

	for i := 0; i < s.GetTimeout(); i++ {
		testErrors = s.Check(namespace)

		if len(testErrors) == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	if len(testErrors) == 0 {
		s.Logger.Log("test step completed", s.String())
	} else {
		s.Logger.Log("test step failed", s.String())
	}

	return testErrors
}

// String implements the string interface, returning the name of the test step.
func (s *Step) String() string {
	return fmt.Sprintf("%d-%s", s.Index, s.Name)
}

// LoadYAML loads the resources from a YAML file for a test step:
// * If the YAML file is called "assert", then it contains objects to
//   add to the test step's list of assertions.
// * If the YAML file is called "errors", then it contains objects that,
//   if seen, mark a test immediately failed.
// * All other YAML files are considered resources to create.
func (s *Step) LoadYAML(file string) error {
	objects, err := testutils.LoadYAML(file)
	if err != nil {
		return fmt.Errorf("loading %s: %s", file, err)
	}

	matches := fileNameRegex.FindStringSubmatch(filepath.Base(file))
	fname := matches[2]

	switch fname {
	case "assert":
		s.Asserts = append(s.Asserts, objects...)
	case "errors":
		s.Errors = append(s.Errors, objects...)
	default:
		if s.Name == "" {
			s.Name = fname
		}
		s.Apply = append(s.Apply, objects...)
	}

	asserts := []runtime.Object{}

	for _, obj := range s.Asserts {
		if obj.GetObjectKind().GroupVersionKind().Kind == "TestAssert" {
			s.Assert = obj.(*kudo.TestAssert)
		} else {
			asserts = append(asserts, obj)
		}
	}

	apply := []runtime.Object{}

	for _, obj := range s.Apply {
		if obj.GetObjectKind().GroupVersionKind().Kind == "TestStep" {
			s.Step = obj.(*kudo.TestStep)
			s.Step.Index = s.Index
			if s.Step.Name != "" {
				s.Name = s.Step.Name
			}
		} else {
			apply = append(apply, obj)
		}
	}

	s.Apply = apply
	s.Asserts = asserts
	return nil
}
