package get

import (
	"errors"
	"fmt"
	"io"

	"github.com/xlab/treeprint"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

type CmdOpts struct {
	Out    io.Writer
	Client *kudo.Client

	Output    output.Type
	Namespace string
}

const (
	All = "all"

	Instances        = "instances"
	Operators        = "operators"
	OperatorVersions = "operatorversions"
)

// Run returns the errors associated with cmd env
func Run(args []string, opts CmdOpts) error {
	if err := opts.Output.Validate(); err != nil {
		return err
	}

	err := validate(args)
	if err != nil {
		return err
	}

	var objs []runtime.Object
	switch args[0] {
	case Instances:
		objs, err = opts.Client.ListInstancesAsRuntimeObject(opts.Namespace)
	case Operators:
		objs, err = opts.Client.ListOperatorsAsRuntimeObject(opts.Namespace)
	case OperatorVersions:
		objs, err = opts.Client.ListOperatorVersionsAsRuntimeObject(opts.Namespace)
	case All:
		return runGetAll(opts)
	}
	if err != nil {
		return fmt.Errorf("failed to retrieve objects: %v", err)
	}

	if opts.Output.IsFormattedOutput() {
		var outObj []interface{}
		for _, o := range objs {
			outObj = append(outObj, o)
		}
		return output.WriteObjects(outObj, opts.Output, opts.Out)
	}

	tree := treeprint.New()

	metadataAccessor := meta.NewAccessor()
	for _, obj := range objs {
		name, err := metadataAccessor.Name(obj)
		if err != nil {
			return fmt.Errorf("failed to retrieve name from %v: %v", obj, err)
		}
		tree.AddBranch(name)
	}
	fmt.Fprintf(opts.Out, "List of current installed %s in namespace %q:\n", args[0], opts.Namespace)
	fmt.Fprintln(opts.Out, tree.String())
	return err
}

func runGetAll(opts CmdOpts) error {
	instances, err := opts.Client.ListInstancesAsRuntimeObject(opts.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get instances")
	}
	operatorversions, err := opts.Client.ListOperatorVersionsAsRuntimeObject(opts.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get operatorversions")
	}
	operators, err := opts.Client.ListOperatorsAsRuntimeObject(opts.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get operators")
	}

	if opts.Output.IsFormattedOutput() {
		var outObj []interface{}
		for _, o := range operators {
			outObj = append(outObj, o)
		}
		for _, o := range operatorversions {
			outObj = append(outObj, o)
		}
		for _, o := range instances {
			outObj = append(outObj, o)
		}
		return output.WriteObjects(outObj, opts.Output, opts.Out)
	}

	return printAllTree(opts, operators, operatorversions, instances)
}

func printAllTree(opts CmdOpts, operators, operatorversions, instances []runtime.Object) error {

	rootTree := treeprint.New()
	for _, o := range operators {
		op, _ := o.(*v1beta1.Operator)
		opTree := rootTree.AddBranch(op.Name)

		for _, ovo := range operatorversions {
			ov, _ := ovo.(*v1beta1.OperatorVersion)
			if ov.Spec.Operator.Name == op.Name {
				ovTree := opTree.AddBranch(ov.Name)

				for _, io := range instances {
					i, _ := io.(*v1beta1.Instance)
					if i.Spec.OperatorVersion.Name == ov.Name {
						ovTree.AddBranch(i.Name)
					}
				}
			}
		}
	}

	fmt.Fprintf(opts.Out, "List of current installed operators including versions and instances in namespace %q:\n", opts.Namespace)
	fmt.Fprintln(opts.Out, rootTree.String())
	return nil

}

func validate(args []string) error {
	if len(args) != 1 {
		return errors.New(`expecting exactly one argument - "instances, operators, operatorversions or all"`)
	}

	switch args[0] {
	case Instances, Operators, OperatorVersions:
		fallthrough
	case All:
		return nil
	default:
		return fmt.Errorf(`expecting one of "instances, operators, operatorversions or all" and not %q`, args[0])
	}
}
