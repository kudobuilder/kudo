package plan

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

type TriggerOptions struct {
	Plan     string
	Instance string
}

// RunTrigger triggers a plan execution
func RunTrigger(options *TriggerOptions, settings *env.Settings) error {
	if options.Instance == "" {
		return errors.New("please choose the instance with '--instance=<instanceName>'")
	}
	if options.Plan == "" {
		return errors.New("please choose the plan name with '--name=<planName>'")
	}

	kc, err := env.GetClient(settings)
	if err != nil {
		return fmt.Errorf("creating kudo client: %w", err)
	}

	err = kc.UpdateInstance(options.Instance, settings.Namespace, nil, nil, &options.Plan)
	if err == nil {
		clog.Printf("Triggered %s plan for %s/%s instance", options.Plan, settings.Namespace, options.Instance)
	}
	return err
}
