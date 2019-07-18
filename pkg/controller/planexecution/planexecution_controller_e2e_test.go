// +build e2e

package planexecution

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Reconcile_Delete_Job tests the planexecution controller with its delete logic for jobs. We expect a job to be
// launched and if the delete boolean is set, immediately deleted right after. We test this by looking at the events
// produced by the controller, in particular the "JobDeletionSuccess" event.
func TestReconcilePlanExecution_Reconcile_Delete_Job(t *testing.T) {

	// pointing the client to a configured k8s cluster
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}

	// creating a random temporary namespace
	namespace := fmt.Sprintf("kudo-test-e2e-%s", rand.String(6))
	// time when we collect the events on which we define if this test failed or succeeded
	timeToCollectEvents := 20
	// the timeout should incorporate network latency and always be higher than timeToCollectEvents
	timeout := 30

	tmpBootstrapFile, err := ioutil.TempFile("", "bootstrap.yaml")
	assert.Nil(t, err)
	defer tmpBootstrapFile.Close()

	ioutil.WriteFile(tmpBootstrapFile.Name(), []byte(`
apiVersion: kudo.k8s.io/v1alpha1
kind: OperatorVersion
metadata:
  name: e2e-operator-1.0
  namespace: `+namespace+`
spec:
  operator:
    name: e2e-operator
    kind: Operator
  version: "1.0"
  parameters:
  - name: REPLICAS
    description: "Number of nginx replicas"
    default: "3"
    displayName: "Replica count"
  - name: PARAM
    description: "Sample parameter"
    default: "before"
  templates:
    deploy.yaml: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: nginx
      spec:
        replicas: {{ .Params.REPLICAS }}
        selector:
          matchLabels:
            app: nginx
        template:
          metadata:
            labels:
              app: nginx
          spec:
            containers:
            - name: nginx
              image: nginx:1.7.9
              ports:
              - containerPort: 80
              env:
              - name: PARAM_ENV
                value: {{ .Params.PARAM }}
    validation.yaml: |
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: sleeper
      spec:
        template:
          metadata:
            name: sleeper
          spec:
            containers:
            - name: sleeper
              image: busybox
              args:
              - /bin/sh
              - -c
              - date; sleep 2
            restartPolicy: OnFailure
  tasks:
    deploy:
      resources:
      - deploy.yaml
    validation:
      resources:
      - validation.yaml
  plans:
    deploy:
      strategy: serial
      phases:
      - name: deploy
        strategy: parallel
        steps:
        - name: deploy
          tasks:
          - deploy
      - name: validation
        strategy: parallel
        steps:
          - name: validation
            tasks:
              - validation
            delete: true
`), 0644)

	tmpNsFile, err := ioutil.TempFile("", "namespace.yaml")
	assert.Nil(t, err)
	defer tmpNsFile.Close()

	ioutil.WriteFile(tmpNsFile.Name(), []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: `+namespace+`
`), 0644)

	tmpInstanceFile, err := ioutil.TempFile("", "instance.yaml")
	assert.Nil(t, err)
	defer tmpInstanceFile.Close()

	ioutil.WriteFile(tmpInstanceFile.Name(), []byte(`
apiVersion: kudo.k8s.io/v1alpha1
kind: Instance
metadata:
  name: e2e-operator
  namespace: `+namespace+`
  labels:
    kudo.dev/operator: e2e-operator
spec:
  operatorVersion:
    name: e2e-operator-1.0
    kind: OperatorVersion
`), 0644)

	// Creating namespace for test environment
	var nsObj runtime.Object
	nsObjs, err := testutils.LoadYAML(tmpNsFile.Name())
	for _, obj := range nsObjs {
		// creating the object
		err := cl.Create(context.TODO(), obj)
		if err != nil {
			t.Logf("Error while creating namespace object: %#v", err)
		}
		nsObj = obj
		assert.Nil(t, err)
	}

	c1 := make(chan string, 1)
	go func() {

		// creating bootstrap environment
		objs, err := testutils.LoadYAML(tmpBootstrapFile.Name())
		assert.Nil(t, err)
		for _, obj := range objs {
			// creating the object
			err := cl.Create(context.TODO(), obj)
			if err != nil {
				t.Logf("Error while creating e2e bootstrap object(s): %#v", err)
			}
		}

		// Give it a little time to settle in
		time.Sleep(2 * time.Second)

		// creating test instance
		objs, err = testutils.LoadYAML(tmpInstanceFile.Name())
		assert.Nil(t, err)
		for _, obj := range objs {
			// creating the object
			err := cl.Create(context.TODO(), obj)
			if err != nil {
				t.Logf("Error while creating instance object(s): %#v", err)
			}
		}

		// give it a little time to run the actual operator so we don't miss out on events
		time.Sleep(time.Duration(timeToCollectEvents) * time.Second)

		// running test logic
		newOpts := client.ListOptions{Namespace: namespace}
		lo := client.UseListOptions(&newOpts)
		eventList := &corev1.EventList{}
		cl.List(context.TODO(), eventList, lo)
		for _, event := range eventList.Items {
			t.Logf("Event Name: %s Event Reason: %s", event.Name, event.Reason)
			if event.Reason == "JobDeletionSuccess" {
				c1 <- "success"
			}
		}

		c1 <- "failed"
	}()
	select {
	case res := <-c1:
		if res == "success" {
			t.Logf("E2E finished successfully")
			// Cleaning up the temporarily created namespace
			assert.Nil(t, cl.Delete(context.TODO(), nsObj))
		}
		if res == "failed" {
			t.Errorf("E2E failed")
			// Cleaning up the temporarily created namespace
			assert.Nil(t, cl.Delete(context.TODO(), nsObj))
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		t.Errorf("E2E timed out after %ds", timeout)
		// Cleaning up the temporarily created namespace
		assert.Nil(t, cl.Delete(context.TODO(), nsObj))
	}
}
