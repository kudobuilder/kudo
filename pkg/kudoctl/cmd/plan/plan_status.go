package plan

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/thoas/go-funk"
	"github.com/xlab/treeprint"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// Options are the configurable options for plans
type StatusOptions struct {
	Out      io.Writer
	Instance string
	Wait     bool
	Output   output.Type
}

// Status runs the plan status command
func Status(options *StatusOptions, settings *env.Settings) error {
	kc, err := env.GetClient(settings)
	if err != nil {
		return err
	}

	return status(kc, options, settings.Namespace)
}

func statusFormatted(kc *kudo.Client, options *StatusOptions, ns string) error {
	instance, err := kc.GetInstance(options.Instance, ns)
	if err != nil {
		return err
	}
	if instance == nil {
		return fmt.Errorf("instance %s/%s does not exist", ns, options.Instance)
	}
	return output.WriteObject(instance.Status, options.Output, options.Out)
}

func status(kc *kudo.Client, options *StatusOptions, ns string) error {

	if options.Output != "" {
		return statusFormatted(kc, options, ns)
	}

	firstPass := true
	start := time.Now()

	// for loop breaks if Wait==false, or when active plan completes (or when user exits process)
	for {
		tree := treeprint.New()

		instance, err := kc.GetInstance(options.Instance, ns)
		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("instance %s/%s does not exist", ns, options.Instance)
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

		getPhaseStrategy := func(s string) kudoapi.Ordering {
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
		// exec on first go, otherwise don't
		if firstPass {
			fmt.Fprintf(options.Out, "Plan(s) for \"%s\" in namespace \"%s\":\n", instance.Name, ns)
		}
		// exec on all loop passes except the first
		if !firstPass {
			height := strings.Count(tree.String(), "\n") + 1
			clearLines(options.Out, height)
		}
		fmt.Fprintln(options.Out, tree.String())
		firstPass = false
		if options.Wait {
			elapsed := time.Since(start)
			clearLine(options.Out)
			fmt.Fprintf(options.Out, "elapsed time %s", elapsed)
		} else {
			break
		}
		done, err := kc.IsInstanceDone(instance, nil)
		if err != nil {
			return err
		}
		if done {
			break
		}
		// freq of updates
		time.Sleep(1 * time.Second)
	}
	return nil
}

// moves terminal cursor up number of lines specified
func moveCursorUp(w io.Writer, lines int) {
	fmt.Fprintf(w, "\033[%dA", lines)
}

// clears the current terminal line
func clearLine(w io.Writer) {
	fmt.Fprint(w, "\u001b[0K\r")
}

// clears multiple terminal lines from current position up to defined height
// useful to clear previous terminal output in order to rewrite to that screen section
func clearLines(w io.Writer, height int) {
	moveCursorUp(w, height)
	for i := 0; i < height; i++ {
		clearLine(w)
		fmt.Fprint(w, "\n")
	}
	moveCursorUp(w, height)
}

func printMessageIfAvailable(s string) string {
	if s != "" {
		return fmt.Sprintf(" (%s)", s)
	}
	return ""
}
