package diagnostics

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/version"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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

	config, _ := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	ns := settings.Namespace
	kc, _ := env.GetClient(settings)
	instance, _ := kc.GetInstance(Options.Instance, ns)
	c, _ := kube.GetKubeClient(settings.KubeConfig)

	byOperator := metav1.ListOptions{
		LabelSelector: labelKudoOperator + "=" + instance.Labels[labelKudoOperator],
	}
	byKudoManager := metav1.ListOptions{
		LabelSelector: "app=" + appKudoManager,
	}

	iResources := &resourceFuncs{c, kc, ns, byOperator, corev1.PodLogOptions{}, instance}
	cResources := &resourceFuncs{c, nil, nsKudoSystem, byKudoManager, corev1.PodLogOptions{}, nil} // no need for kudo and instance

	// TODO: use more meaningful variable names
	// TODO: use Builder
	dc := resourceListCollector{getResources: iResources.deployments()}
	pc := resourceListCollector{getResources: iResources.pods()}
	ec := resourceListCollector{getResources: iResources.events()}
	sc := resourceListCollector{getResources: iResources.services()}
	sac := resourceListCollector{getResources: iResources.serviceAccounts(&pc)}
	rsc := resourceListCollector{getResources: iResources.replicaSets()}
	ovc := resourceCollector{getResource: iResources.operatorVersions()}
	oc := resourceCollector{getResource: iResources.operators(&ovc)}
	lc := logCollector{getLogs: iResources.logs(&pc)}
	cpc := resourceListCollector{getResources: cResources.pods()}
	ssc := resourceListCollector{getResources: cResources.statefulSets()}
	rbc := resourceListCollector{getResources: cResources.roleBindings(&sac)}
	crbc := resourceListCollector{getResources: cResources.clusterRoleBindings(&sac)}
	rc := resourceListCollector{getResources: cResources.roles(&rbc)}
	crc := resourceListCollector{getResources: cResources.clusterRoles(&crbc)}
	clc := logCollector{getLogs: cResources.logs(&cpc)}
	verc := dumpingCollector{s: version.Get()}
	setc := dumpingCollector{s: *settings}

	// TODO: either make order irrelevant or add order validation
	collectors := []Collector{&dc, &pc, &ec, &sc, &sac, &rsc, &ovc, &oc, &cpc, &ssc, &rbc, &crbc, &rc, &crc, &lc, &clc, &verc, &setc}
	var describers []Collector
	for _, c := range collectors {
		switch rc := c.(type) {
		case *resourceCollector:
			describers = append(describers, &describeCollector{getDescribe: description(rc, config)})
		case *resourceListCollector:
			describers = append(describers, &describeListCollector{getDescribes: descriptions(rc, config)})
		}
	}
	collectors = append(collectors, describers...)

	var errors *multiError
	for _, c := range collectors {
		err := c.Collect(fileWriter)
		errors = appendError(errors, err)
	}
	if errors == nil {
		return nil
	}

	w, _ := fileWriter(errors)
	fmt.Fprint(w, errors)
	return errors
}
