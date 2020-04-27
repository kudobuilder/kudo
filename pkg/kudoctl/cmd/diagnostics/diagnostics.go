package diagnostics

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"
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

type Collector interface {
	Collect(f writerFactory) error
}

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

	bu := NewBuilder()
	bu.
		AddResource(ir.instance(),
			nameExtractorWithKey{"OperatorVersion", operatorVersionForInstance}).
		AddResource(ir.operatorVersion(bu.GetName("OperatorVersion")),
			nameExtractorWithKey{"OperatorName", operatorForOperatorVersion}).
		AddResource(ir.operator(bu.GetName("OperatorName"))).
		AddResources(ir.deployments()).
		AddResources(ir.pods(),
			namesExtractorWithKey{key: "InstanceServiceAccountNames", extractor: serviceAccountsForPods},
			namesExtractorWithKey{key: "InstancePodNames", extractor: pods},
		).
		AddResources(ir.events()).   // TODO: filter and group events per InvolvedObjects
		AddResources(ir.services()). // TODO: should not only match by label but also filter by services' selectors!
		AddResources(ir.serviceAccounts(bu.GetNames("InstanceServiceAccountNames"))).
		AddResources(ir.clusterRoleBindings(bu.GetNames("InstanceServiceAccountNames")),
			namesExtractorWithKey{"InstanceClusterRoleNames", clusterRolesForBindings}).
		AddResources(ir.roleBindings(bu.GetNames("InstanceServiceAccountNames")),
			namesExtractorWithKey{"InstanceRoleNames", rolesForBindings}).
		AddResources(ir.replicaSets()).
		AddResources(ir.statefulSets())
	bu.
		AddResources(cr.pods(),
			namesExtractorWithKey{key: "KudoServiceAccountNames", extractor: serviceAccountsForPods},
			namesExtractorWithKey{key: "KudoPodNames", extractor: pods},
		).
		AddResources(cr.statefulSets()).
		AddResources(cr.services()).
		AddResources(cr.serviceAccounts(bu.GetNames("KudoServiceAccountNames"))).
		AddResources(cr.clusterRoleBindings(bu.GetNames("KudoServiceAccountNames")),
			namesExtractorWithKey{"KudoClusterRoleNames", clusterRolesForBindings}).
		AddResources(cr.roleBindings(bu.GetNames("KudoServiceAccountNames")),
			namesExtractorWithKey{"KudoRoleNames", rolesForBindings}).
		AddResources(cr.clusterRoles(bu.GetNames("KudoClusterRoleNames"))).
		AddResources(cr.roles(bu.GetNames("KudoRoleNames")))

	collectors := bu.Build()
	collectors = append(collectors, &logCollector{getLogs: ir.logs(bu.GetNames("InstancePodNames"))})
	collectors = append(collectors, &logCollector{getLogs: cr.logs(bu.GetNames("KudoPodNames"))})
	collectors = append(collectors, &dumpingCollector{s: version.Get()})
	collectors = append(collectors, &dumpingCollector{s: *settings})

	//var describers []Collector
	//config, _ := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	//for _, c := range collectors {
	//	switch rc := c.(type) {
	//	case *resourceCollector:
	//		describers = append(describers, &describeCollector{getDescribe: description(rc, config)})
	//	case *resourceListCollector:
	//		describers = append(describers, &describeListCollector{getDescribes: descriptions(rc, config)})
	//	}
	//}
	//collectors = append(collectors, describers...)

	var errors *MultiError
	for i, c := range collectors {
		fmt.Println("collector #", i)
		err := c.Collect(fileWriter)
		errors = AppendError(errors, err)
	}
	if errors == nil {
		return nil
	}

	w, _ := fileWriter(errors)
	fmt.Fprint(w, errors)
	return errors
}
