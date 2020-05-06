package diagnostics

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	//"github.com/kudobuilder/kudo/pkg/version"
	"github.com/spf13/cobra"
)

const (
	nsKudoSystem      = "kudo-system"
	labelKudoOperator = "kudo.dev/operator"
	appKudoManager    = "kudo-manager"
)

var Options = struct {
	Instance string
}{}

func Collect(cmd *cobra.Command, settings *env.Settings) error {
	fmt.Println("Collecting diagnostics")

	ir, err := NewInstanceResources(Options.Instance, settings)
	if err != nil {
		return nil
	}
	cr, err := NewKudoResources(settings)
	if err != nil {
		return nil
	}
	irf := newResourceProviderFactory(ir)
	crf := newResourceProviderFactory(cr)

	b := NewOtherBuilder(irf, DefaultNameProviders())
	kinds := []string{"instance", "operatorversion", "operator", "pod", "statefulset", "replicaset", "event",
		"deployment", "service", "serviceaccount", "role", "clusterrole"}
	for _, kind := range kinds {
		b.AddResource(kind)
	}
	cache := b.Collect()
	tree := BuildPrintableTree(cache, DefaultAttachmentRules(), kinds)
	err = printTree(tree, "")
	fmt.Println(err)

	b = NewOtherBuilder(crf, DefaultNameProviders())
	for _, kind := range kinds {
		b.AddResource(kind)
	}
	cache = b.Collect()
	tree = BuildPrintableTree(cache, DefaultKudoAttachmentRules(), kinds)
	err = printTree(tree, "kudo")
	fmt.Println(err)

	return err
}
