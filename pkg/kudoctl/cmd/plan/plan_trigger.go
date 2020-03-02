package plan

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

type TriggerOptions struct {
	Plan     string
	Instance string
}

// RunHistory runs the plan history command
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

	return kc.UpdateInstance(options.Instance, settings.Namespace, nil, nil, &options.Plan)
}
