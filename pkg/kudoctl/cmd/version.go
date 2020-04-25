package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/version"
)

var (
	versionExample = `  # Print the current installed KUDO package version
  kubectl kudo version`
)

// newVersionCmd returns a new initialized instance of the version sub command
func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the current KUDO package version.",
		Long:    `Print the current installed KUDO package version.`,
		Example: versionExample,
		RunE:    VersionCmd,
	}

	return versionCmd
}

// VersionCmd performs the version sub command
func VersionCmd(cmd *cobra.Command, args []string) error {
	kudoVersion := version.Get()
	fmt.Printf("KUDO Version: %s\n", fmt.Sprintf("%#v", kudoVersion))

	// Print the Controller Version
	fmt.Printf("KUDO Controller Version: %s\n", GetControllerVersion())

	return nil
}

// GetControllerVersion
func GetControllerVersion() string {

	controllerVersion := "<Controller Not Running>"

	client, err := kube.GetKubeClient(Settings.KubeConfig)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire client")
		return controllerVersion
	}

	statefulsets, err := client.KubeClient.AppsV1().StatefulSets("").List(metav1.ListOptions{LabelSelector: "app=kudo-manager"})
	if err != nil {
		return controllerVersion
	}

	for _, d := range statefulsets.Items {
		controllerVersion = d.Spec.Template.Spec.Containers[0].Image
	}

	return controllerVersion
}
