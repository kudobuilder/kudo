package plan

import (
	"fmt"
	"sort"

	"github.com/thoas/go-funk"
	"github.com/xlab/treeprint"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// DefaultStatusOptions provides the default options for plan status
var DefaultStatusOptions = &Options{}

// Status runs the plan status command
func Status(options *Options, settings *env.Settings) error {
	kc, err := env.GetClient(settings)
	if err != nil {
		return err
	}

	return status(kc, options, settings.Namespace)
}

func status(kc *kudo.Client, options *Options, ns string) error {
	tree := treeprint.New()

	instance, err := kc.GetInstance(options.Instance, ns)
	if err != nil {
		return err
	}
	if instance == nil {
		return fmt.Errorf("Instance %s/%s does not exist", ns, options.Instance)
	}

	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, ns)
	if err != nil {
		return err
	}
	if ov == nil {
		return fmt.Errorf("OperatorVersion %s from instance %s/%s does not exist", instance.Spec.OperatorVersion.Name, ns, options.Instance)
	}

	lastPlanStatus := instance.GetLastExecutedPlanStatus()

	if lastPlanStatus == nil {
		fmt.Fprintf(options.Out, "No plan ever run for instance - nothing to show for instance %s\n", instance.Name)
		return nil
	}

	getPhaseStrategy := func(s string) v1beta1.Ordering {
		for _, plan := range ov.Spec.Plans {
			for _, phase := range plan.Phases {
				if phase.Name == s {
					return phase.Strategy
				}
			}
		}
		return ""
	}

	rootDisplay := fmt.Sprintf("%s (Operator-Version: \"%s\" Active-Plan: \"%s\")", instance.Name, instance.Spec.OperatorVersion.Name, lastPlanStatus.Name)
	rootBranchName := tree.AddBranch(rootDisplay)

	plans, _ := funk.Keys(ov.Spec.Plans).([]string)
	sort.Strings(plans)

	for _, plan := range plans {
		if plan == lastPlanStatus.Name {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [%s]%s", plan, ov.Spec.Plans[plan].Strategy, lastPlanStatus.Status, printMessageIfAvailable(lastPlanStatus.Message))
			if lastPlanStatus.LastUpdatedTimestamp != nil {
				planDisplay = fmt.Sprintf("%s, last updated %s", planDisplay, lastPlanStatus.LastUpdatedTimestamp.Format("2006-01-02 15:04:05"))
			}
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range lastPlanStatus.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s (%s strategy) [%s]%s", phase.Name, getPhaseStrategy(phase.Name), phase.Status, printMessageIfAvailable(phase.Message))
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range phase.Steps {
					stepsDisplay := fmt.Sprintf("Step %s [%s]%s", steps.Name, steps.Status, printMessageIfAvailable(steps.Message))
					phaseBranchName.AddBranch(stepsDisplay)
				}
			}
		} else {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [NOT ACTIVE]", plan, ov.Spec.Plans[plan].Strategy)
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range ov.Spec.Plans[plan].Phases {
				phaseDisplay := fmt.Sprintf("Phase %s (%s strategy) [NOT ACTIVE]", phase.Name, phase.Strategy)
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range phase.Steps {
					stepDisplay := fmt.Sprintf("Step %s [NOT ACTIVE]", steps.Name)
					phaseBranchName.AddBranch(stepDisplay)
				}
			}
		}
	}

	fmt.Fprintf(options.Out, "Plan(s) for \"%s\" in namespace \"%s\":\n", instance.Name, ns)
	fmt.Fprintln(options.Out, tree.String())

	return nil
}

func printMessageIfAvailable(s string) string {
	if s != "" {
		return fmt.Sprintf(" (%s)", s)
	}
	return ""
}
