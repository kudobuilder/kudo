package test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

var fileNameRegex = regexp.MustCompile(`^(\d+-)?([^.]+)(.yaml)?$`)

// A Step contains the name of the test step, its index in the test,
// and all of the test step's settings (including objects to apply and assert on).
type Step struct {
	Name  string
	Index int

	Dir string

	Step   *kudo.TestStep
	Assert *kudo.TestAssert

	Asserts []runtime.Object
	Apply   []runtime.Object
	Errors  []runtime.Object

	Timeout int

	Client          func(forceNew bool) (client.Client, error)
	DiscoveryClient func() (discovery.DiscoveryInterface, error)

	Logger testutils.Logger
}

// Clean deletes all resources defined in the Apply list.
func (s *Step) Clean(namespace string) error {
	cl, err := s.Client(false)
	if err != nil {
		return err
	}

	dClient, err := s.DiscoveryClient()
	if err != nil {
		return err
	}

	for _, obj := range s.Apply {
		_, _, err := testutils.Namespaced(dClient, obj, namespace)
		if err != nil {
			return err
		}

		if err := cl.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// DeleteExisting deletes any resources in the TestStep.Delete list prior to running the tests.
func (s *Step) DeleteExisting(namespace string) error {
	cl, err := s.Client(false)
	if err != nil {
		return err
	}

	dClient, err := s.DiscoveryClient()
	if err != nil {
		return err
	}

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

		_, objNs, err := testutils.Namespaced(dClient, obj, objNs)
		if err != nil {
			return err
		}

		if ref.Name == "" {
			u := &unstructured.UnstructuredList{}
			u.SetGroupVersionKind(gvk)

			listOptions := []client.ListOption{}

			if ref.Labels != nil {
				listOptions = append(listOptions, client.MatchingLabels(ref.Labels))
			}

			if objNs != "" {
				listOptions = append(listOptions, client.InNamespace(objNs))
			}

			err := cl.List(context.TODO(), u, listOptions...)
			if err != nil {
				return fmt.Errorf("listing matching resources: %w", err)
			}

			for index := range u.Items {
				toDelete = append(toDelete, &u.Items[index])
			}
		} else {
			// Otherwise just append the object specified.
			toDelete = append(toDelete, obj.DeepCopyObject())
		}
	}

	for _, obj := range toDelete {
		err := cl.Delete(context.TODO(), obj.DeepCopyObject())
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	// Wait for resources to be deleted.
	return wait.PollImmediate(100*time.Millisecond, s.GetTimeout()*time.Second, func() (done bool, err error) {
		for _, obj := range toDelete {
			err = cl.Get(context.TODO(), testutils.ObjectKey(obj), obj.DeepCopyObject())
			if err == nil || !k8serrors.IsNotFound(err) {
				return false, err
			}
		}

		return true, nil
	})
}

// Create applies all resources defined in the Apply list.
func (s *Step) Create(namespace string) []error {
	cl, err := s.Client(true)
	if err != nil {
		return []error{err}
	}

	dClient, err := s.DiscoveryClient()
	if err != nil {
		return []error{err}
	}

	errors := []error{}

	for _, obj := range s.Apply {
		_, _, err := testutils.Namespaced(dClient, obj, namespace)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if updated, err := testutils.CreateOrUpdate(context.TODO(), cl, obj, true); err != nil {
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
	timeout := s.Timeout
	if s.Assert != nil && s.Assert.Timeout != 0 {
		timeout = s.Assert.Timeout
	}
	return timeout
}

func list(cl client.Client, gvk schema.GroupVersionKind, namespace string) ([]unstructured.Unstructured, error) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	listOptions := []client.ListOption{}
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	if err := cl.List(context.TODO(), &list, listOptions...); err != nil {
		return []unstructured.Unstructured{}, err
	}

	return list.Items, nil
}

// CheckResource checks if the expected resource's state in Kubernetes is correct.
func (s *Step) CheckResource(expected runtime.Object, namespace string) []error {
	cl, err := s.Client(false)
	if err != nil {
		return []error{err}
	}

	dClient, err := s.DiscoveryClient()
	if err != nil {
		return []error{err}
	}

	testErrors := []error{}

	name, namespace, err := testutils.Namespaced(dClient, expected, namespace)
	if err != nil {
		return append(testErrors, err)
	}

	gvk := expected.GetObjectKind().GroupVersionKind()

	actuals := []unstructured.Unstructured{}

	if name != "" {
		actual := unstructured.Unstructured{}
		actual.SetGroupVersionKind(gvk)

		err = cl.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &actual)

		actuals = append(actuals, actual)
	} else {
		actuals, err = list(cl, gvk, namespace)
		if len(actuals) == 0 {
			testErrors = append(testErrors, fmt.Errorf("no resources matched of kind: %s", gvk.String()))
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
			diff, diffErr := testutils.PrettyDiff(expected, &actual)
			if diffErr == nil {
				tmpTestErrors = append(tmpTestErrors, fmt.Errorf(diff))
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

// CheckResourceAbsent checks if the expected resource's state is absent in Kubernetes.
func (s *Step) CheckResourceAbsent(expected runtime.Object, namespace string) error {
	cl, err := s.Client(false)
	if err != nil {
		return err
	}

	dClient, err := s.DiscoveryClient()
	if err != nil {
		return err
	}

	name, namespace, err := testutils.Namespaced(dClient, expected, namespace)
	if err != nil {
		return err
	}

	gvk := expected.GetObjectKind().GroupVersionKind()

	var actuals []unstructured.Unstructured

	if name != "" {
		actual := unstructured.Unstructured{}
		actual.SetGroupVersionKind(gvk)

		if err := cl.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &actual); err != nil {
			if k8serrors.IsNotFound(err) {
				return nil
			}

			return err
		}

		actuals = []unstructured.Unstructured{actual}
	} else {
		actuals, err = list(cl, gvk, namespace)
		if err != nil {
			return err
		}
	}

	expectedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(expected)
	if err != nil {
		return err
	}

	for _, actual := range actuals {
		if err := testutils.IsSubset(expectedObj, actual.UnstructuredContent()); err == nil {
			return fmt.Errorf("resource matched of kind: %s", gvk.String())
		}
	}

	return nil
}

// Check checks if the resources defined in Asserts and Errors are in the correct state.
func (s *Step) Check(namespace string) []error {
	testErrors := []error{}

	for _, expected := range s.Asserts {
		testErrors = append(testErrors, s.CheckResource(expected, namespace)...)
	}

	for _, expected := range s.Errors {
		if testError := s.CheckResourceAbsent(expected, namespace); testError != nil {
			testErrors = append(testErrors, testError)
		}
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

	testErrors := []error{}

	if s.Step != nil {
		if errors := testutils.RunCommands(s.Logger, namespace, "", s.Step.Commands, s.Dir); errors != nil {
			testErrors = append(testErrors, errors...)
		}

		if errors := testutils.RunKubectlCommands(s.Logger, namespace, s.Step.Kubectl, s.Dir); errors != nil {
			testErrors = append(testErrors, errors...)
		}
	}

	testErrors = append(testErrors, s.Create(namespace)...)

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
