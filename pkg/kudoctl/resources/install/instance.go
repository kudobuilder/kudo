package install

import (
	"errors"
	"fmt"
	"time"

	pollwait "k8s.io/apimachinery/pkg/util/wait"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// Instance installs a KUDO instance to a cluster.
// It returns an error if the namespace already contains an instance with the same name.
func Instance(client *kudo.Client, instance *kudoapi.Instance) error {
	existingInstance, err := client.GetInstance(instance.Name, instance.Namespace)
	if err != nil {
		return fmt.Errorf("failed to verify existing instance: %v", err)
	}

	if existingInstance != nil {
		return clog.Errorf(
			"cannot install instance '%s' because an instance of that name already exists in namespace %s",
			instance.Name,
			instance.Namespace)
	}

	if _, err := client.InstallInstanceObjToCluster(instance, instance.Namespace); err != nil {
		return fmt.Errorf(
			"failed to install instance %s/%s: %v",
			instance.Namespace,
			instance.Name,
			err)
	}

	clog.Printf("instance %s/%s created", instance.Namespace, instance.Name)
	return nil
}

// WaitForInstance waits for an amount of time for an instance to be "complete".
func WaitForInstance(client *kudo.Client, instance *kudoapi.Instance, timeout time.Duration) error {
	err := client.WaitForInstance(instance.Name, instance.Namespace, nil, timeout)
	if errors.Is(err, pollwait.ErrWaitTimeout) {
		clog.Printf("timeout waiting for instance %s/%s", instance.Namespace, instance.Name)
	}
	if err != nil {
		return fmt.Errorf("failed to wait on instance %s/%s: %v", instance.Namespace, instance.Name, err)
	}

	clog.Printf("instance %s/%s ready", instance.Namespace, instance.Name)
	return nil
}
