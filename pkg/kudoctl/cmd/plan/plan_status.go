package plan

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	"github.com/xlab/treeprint"

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

	lastPlanStatus := getLastExecutedPlanStatus(instance)

	if lastPlanStatus == nil {
		fmt.Fprintf(options.Out, "No plan ever run for instance - nothing to show for instance %s\n", instance.Name)
		return nil
	}

	rootDisplay := fmt.Sprintf("%s (Operator-Version: \"%s\" Active-Plan: \"%s\")", instance.Name, instance.Spec.OperatorVersion.Name, lastPlanStatus.Name)
	rootBranchName := tree.AddBranch(rootDisplay)

	for name, plan := range ov.Spec.Plans {
		if name == lastPlanStatus.Name {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [%s]%s", name, plan.Strategy, lastPlanStatus.Status, printMessageIfAvailable(lastPlanStatus.Message))
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range lastPlanStatus.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s [%s]%s", phase.Name, phase.Status, printMessageIfAvailable(phase.Message))
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range phase.Steps {
					stepsDisplay := fmt.Sprintf("Step %s [%s]%s", steps.Name, steps.Status, printMessageIfAvailable(steps.Message))
					phaseBranchName.AddBranch(stepsDisplay)
				}
			}
		} else {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [NOT ACTIVE]", name, plan.Strategy)
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range plan.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s (%s strategy) [NOT ACTIVE]", phase.Name, phase.Strategy)
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range plan.Phases {
					stepsDisplay := fmt.Sprintf("Step %s (%s strategy) [NOT ACTIVE]", steps.Name, steps.Strategy)
					stepBranchName := phaseBranchName.AddBranch(stepsDisplay)
					for _, step := range steps.Steps {
						stepDisplay := fmt.Sprintf("%s [NOT ACTIVE]", step.Name)
						stepBranchName.AddBranch(stepDisplay)
					}
				}
			}
		}
	}

	fmt.Fprintf(options.Out, "Plan(s) for \"%s\" in namespace \"%s\":\n", instance.Name, ns)
	fmt.Fprintln(options.Out, tree.String())

	return nil
}

// GetPlanInProgress returns plan status of currently active plan or nil if no plan is running
func getPlanInProgress(i *v1beta1.Instance) *v1beta1.PlanStatus {
	for _, p := range i.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
}

// GetLastExecutedPlanStatus returns status of plan that is currently running, if there is one running
// if no plan is running it looks for last executed plan based on timestamps
func getLastExecutedPlanStatus(i *v1beta1.Instance) *v1beta1.PlanStatus {
	if noPlanEverExecuted(i) {
		return nil
	}
	activePlan := getPlanInProgress(i)
	if activePlan != nil {
		return activePlan
	}
	var lastExecutedPlan *v1beta1.PlanStatus
	for n := range i.Status.PlanStatus {
		p := i.Status.PlanStatus[n]
		if p.Status == v1beta1.ExecutionNeverRun {
			continue // only interested in plans that run
		}
		if lastExecutedPlan == nil {
			lastExecutedPlan = &p // first plan that was run and we're iterating over
		} else if wasRunAfter(p, *lastExecutedPlan) {
			lastExecutedPlan = &p // this plan was run after the plan we have chosen before
		}
	}
	return lastExecutedPlan
}

// NoPlanEverExecuted returns true is this is new instance for which we never executed any plan
func noPlanEverExecuted(instance *v1beta1.Instance) bool {
	for _, p := range instance.Status.PlanStatus {
		if p.Status != v1beta1.ExecutionNeverRun {
			return false
		}
	}
	return true
}

// wasRunAfter returns true if p1 was run after p2
func wasRunAfter(p1 v1beta1.PlanStatus, p2 v1beta1.PlanStatus) bool {
	if p1.Status == v1beta1.ExecutionNeverRun || p2.Status == v1beta1.ExecutionNeverRun {
		return false
	}
	return p1.LastFinishedRun.Time.After(p2.LastFinishedRun.Time)
}

func printMessageIfAvailable(s string) string {
	if s != "" {
		return fmt.Sprintf(" (%s)", s)
	}
	return ""
}
