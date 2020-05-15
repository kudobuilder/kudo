package cmd

import (
	"errors"
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
	controllerVersion, err := GetControllerVersion()
	if err != nil {
		fmt.Printf("KUDO Controller Version: %s\n", controllerVersion)
	} else {
		fmt.Printf("KUDO Controller Version: %#v\n", err)
	}

	return nil
}

// GetControllerVersion
func GetControllerVersion() (string, error) {

	controllerVersion := ""

	client, err := kube.GetKubeClient(Settings.KubeConfig)
	clog.V(3).Printf("Acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("Failed to acquire kudo client")
		return "", errors.New("<Failed to acquire kudo client>")
	}

	statefulsets, err := client.KubeClient.AppsV1().StatefulSets("").List(metav1.ListOptions{LabelSelector: "app=kudo-manager"})
	clog.V(3).Printf("List statefulsets and filter kudo-manager")
	if err != nil {
		clog.V(3).Printf("Failed to list kudo-manager statefulset")
		return "", errors.New("<Error: Failed to list kudo-manager statefulset>")
	}

	for _, d := range statefulsets.Items {
		controllerVersion = d.Spec.Template.Spec.Containers[0].Image
	}

	return controllerVersion, nil
}
