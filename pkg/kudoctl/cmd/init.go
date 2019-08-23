package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/env/kudohome"
	manInit "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/init"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const initDesc = `
This command installs KUDO Manager onto your Kubernetes Cluster and sets up local configuration in $KUDO_HOME (default ~/.kudo/).

As with the rest of the KUDO commands, 'kudo init' discovers Kubernetes clusters
by reading $KUBECONFIG (default '~/.kube/config') and using the default context.

To set up just a local environment, use '--client-only'. That will configure
$KUDO_HOME, but not attempt to connect to a Kubernetes cluster and install the KUDO Manager.

When installing KUDO manager, 'kudo init' will attempt to install the latest released
version. You can specify an alternative image with '--kudo-image'.

To dump a manifest containing the KUDO deployment YAML, combine the
'--dry-run' and '--debug' flags.
`

type initCmd struct {
	out        io.Writer
	fs         afero.Fs
	image      string
	clientOnly bool
	dryRun     bool
	home       kudohome.Home
	wait       bool
}

func newInitCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	i := &initCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize KUDO on both client and server",
		Long:  initDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("this command does not accept arguments")
			}
			i.home = Settings.Home

			return i.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&i.image, "kudo-image", "i", "", "Override KUDO manager image")
	f.BoolVarP(&i.clientOnly, "client-only", "c", false, "If set does not install KUDO manager")
	f.BoolVar(&i.dryRun, "dry-run", false, "Do not install local or remote")
	f.BoolVar(&i.wait, "wait", false, "Block until KUDO manager is running and ready to receive requests")

	return cmd
}

// run initializes local config and installs KUDO manager to Kubernetes cluster.
func (i *initCmd) run() error {

	//TODO (kensipe): 1. versioning image
	//TODO (kensipe): 2. how to print crds
	//TODO (kensipe): 3. debug print crd on deploy

	//todo: write manifest file
	//todo: install manifest file
	//todo: use specified image
	if i.dryRun {
		opts := manInit.NewOptions()
		mans, err := manInit.PrereqManifests(opts)
		if err != nil {
			return err
		}

		crd, err := manInit.CrdManifests()
		if err != nil {
			return err
		}

		deploy, err := manInit.DeploymentManifests(opts)
		if err != nil {
			return err
		}

		mans = append(mans, crd...)
		mans = append(mans, deploy...)
		i.YAMLWriter(i.out, mans)
		return nil
	}

	// initialize client
	if err := initialize(i.home, i.out, Settings); err != nil {
		return fmt.Errorf("error initializing: %s", err)
	}
	fmt.Fprintf(i.out, "$KUDO_HOME has been configured at %s.\n", Settings.Home)

	// initialize server
	if !i.clientOnly {
		_, err := getKubeClient(Settings.KubeConfig)
		if err != nil {
			return fmt.Errorf("could not get kubernetes client: %s", err)
		}
		fmt.Fprintln(i.out, "server initialization not implemented yet!")
	} else {
		fmt.Fprintln(i.out, "Not installing KUDO manager due to 'client-only' flag having been set")
	}

	return nil
}

func (i *initCmd) YAMLWriter(w io.Writer, manifests []string) error {
	for _, manifest := range manifests {
		if _, err := fmt.Fprintln(w, "---"); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(w, manifest); err != nil {
			return err
		}
	}

	// YAML ending document boundary marker
	_, err := fmt.Fprintln(w, "...")
	return err
}

func initialize(home kudohome.Home, w io.Writer, settings env.Settings) error {
	//todo: ensure the home dir is created using settings
	return nil
}

func getKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := configForContext(kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return client, nil
}

func configForContext(kubeconfig string) (*rest.Config, error) {
	config, err := kube.GetConfig(kubeconfig).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config using configuration %q: %s", kubeconfig, err)
	}
	return config, nil
}
