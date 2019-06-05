package delete

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// CmdErrorProcessor returns the errors associated with cmd env
func CmdErrorProcessor(cmd *cobra.Command, args []string) error {
	// This makes --kubeconfig flag optional
	if _, err := cmd.Flags().GetString("kubeconfig"); err != nil {
		return fmt.Errorf("get flag: %+v", err)
	}

	if err := check.ValidateKubeConfigPath(); err != nil {
		return errors.Wrap(err, "could not check kubeconfig path")
	}

	if len(args) != 1 {
		return errors.New("you must provide exactly one framework instance to delete")
	}

	kudoclient, err := kudo.NewKudoClient()
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	err = kudoclient.DeleteInstanceFromCluster(args[0])
	if apierrors.IsNotFound(err) {
		return errors.Wrap(err, "framework instance not found")
	}

	if err != nil {
		return errors.Wrap(err, "could not delete framework instance")
	}

	return nil
}
