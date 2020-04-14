package plan

import (
	"errors"
	"fmt"
	"io"
	"time"

	pollwait "k8s.io/apimachinery/pkg/util/wait"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// Options are the configurable options for plans
type WaitOptions struct {
	Out      io.Writer
	Instance string
	WaitTime int64
}

// Status runs the plan status command
func Wait(options *WaitOptions, settings *env.Settings) error {
	kc, err := env.GetClient(settings)
	if err != nil {
		return err
	}
	//return status(kc, options, settings.Namespace)
	return wait(kc, options, settings.Namespace)
}

func wait(kc *kudo.Client, options *WaitOptions, ns string) error {
	instance, err := kc.GetInstance(options.Instance, ns)
	if err != nil {
		return err
	}
	if instance == nil {
		return fmt.Errorf("instance %s/%s does not exist", ns, options.Instance)
	}

	planStatus := instance.GetLastExecutedPlanStatus()
	if planStatus == nil {
		return fmt.Errorf("instance %s/%s does not have an active plan", ns, options.Instance)
	}

	fmt.Fprintf(options.Out, "waiting on instance %s/%s with plan %q\n", ns, options.Instance, planStatus.Name)
	err = kc.WaitForInstance(options.Instance, ns, nil, time.Duration(options.WaitTime)*time.Second)
	if errors.Is(err, pollwait.ErrWaitTimeout) {
		_, _ = fmt.Fprintf(options.Out, "timeout waiting for instance %s/%s on plan %q\n", ns, options.Instance, planStatus.Name)
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(options.Out, "instance %s/%s plan %q finished\n", ns, options.Instance, planStatus.Name)
	return nil
}
