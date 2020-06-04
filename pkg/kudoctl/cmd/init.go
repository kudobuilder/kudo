package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/setup"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const (
	initDesc = `
This command installs KUDO onto your Kubernetes cluster and sets up local configuration in $KUDO_HOME (default ~/.kudo/).

As with the rest of the KUDO commands, 'kudo init' discovers Kubernetes clusters by reading 
$KUBECONFIG (default '~/.kube/config') and using the default context.

To set up just a local environment, use '--client-only'. That will configure $KUDO_HOME, but not attempt to connect 
to a Kubernetes cluster and install KUDO.

When installing KUDO, 'kudo init' will attempt to install the latest released
version. You can specify an alternative image with '--kudo-image' which is the fully qualified image name replacement 
or '--version' which will replace the version designation on the standard image.

To dump a manifest containing the KUDO deployment YAML, combine the '--dry-run' and '--output=yaml' flags.

Running 'kudo init' on server-side is idempotent - it skips manifests already applied to the cluster in previous runs
and finishes with success if KUDO is already installed.
`
	initExample = `  # yaml output
  kubectl kudo init --dry-run --output yaml
  # waiting for KUDO to be installed to the cluster
  kubectl kudo init --wait
  # set up KUDO in your local environment only ($KUDO_HOME)
  kubectl kudo init --client-only
  # set up KUDO in your local environment only (non default $KUDO_HOME)
  kubectl kudo init --client-only --home /opt/home2
  # install kudo crds only
  kubectl kudo init --crd-only
  # delete crds
  kubectl kudo init --crd-only --dry-run --output yaml | kubectl delete -f -
  # pass existing serviceaccount 
  kubectl kudo init --service-account testaccount
  # install kudo using self-signed CA bundle for the webhooks (for testing and development)
  kubectl kudo init --unsafe-self-signed-webhook-ca
`
)

type initCmd struct {
	out                 io.Writer
	errOut              io.Writer
	fs                  afero.Fs
	image               string
	imagePullPolicy     string
	dryRun              bool
	output              string
	version             string
	ns                  string
	serviceAccount      string
	wait                bool
	timeout             int64
	clientOnly          bool
	crdOnly             bool
	home                kudohome.Home
	client              *kube.Client
	selfSignedWebhookCA bool
}

func newInitCmd(fs afero.Fs, out io.Writer, errOut io.Writer, client *kube.Client) *cobra.Command {
	i := &initCmd{fs: fs, out: out, errOut: errOut, client: client}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialize KUDO on both the client and server",
		Long:    initDesc,
		Example: initExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("this command does not accept arguments")
			}
			if err := i.validate(cmd.Flags()); err != nil {
				return err
			}
			i.home = Settings.Home
			i.ns = Settings.OverrideDefault(cmd.Flags(), "namespace", "kudo-system")
			clog.V(8).Printf("init cmd %v", i)
			return i.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&i.clientOnly, "client-only", "c", false, "If set does not install KUDO on the server")
	f.StringVarP(&i.image, "kudo-image", "i", "", "Override KUDO controller image and/or version")
	f.StringVarP(&i.imagePullPolicy, "kudo-image-pull-policy", "", "", "Override KUDO controller image pull policy")
	f.StringVarP(&i.version, "version", "", "", "Override KUDO controller version of the KUDO image")
	f.StringVarP(&i.output, "output", "o", "", "Output format")
	f.BoolVar(&i.dryRun, "dry-run", false, "Do not install local or remote")
	f.BoolVar(&i.crdOnly, "crd-only", false, "Add only KUDO CRDs to your cluster")
	f.BoolVarP(&i.wait, "wait", "w", false, "Block until KUDO manager is running and ready to receive requests")
	f.Int64Var(&i.timeout, "wait-timeout", 300, "Wait timeout to be used")
	f.StringVarP(&i.serviceAccount, "service-account", "", "", "Override for the default serviceAccount kudo-manager")
	f.BoolVar(&i.selfSignedWebhookCA, "unsafe-self-signed-webhook-ca", false, "Use self-signed CA bundle (for testing only) for the webhooks")

	return cmd
}

func (initCmd *initCmd) validate(flags *flag.FlagSet) error {
	// we do not allow the setting of image and version!
	if initCmd.image != "" && initCmd.version != "" {
		return errors.New("specify either 'kudo-image' or 'version', not both")
	}
	if initCmd.clientOnly {
		if initCmd.image != "" || initCmd.version != "" || initCmd.output != "" || initCmd.crdOnly || initCmd.wait {
			return errors.New("you cannot use image, version, output, crd-only and wait flags with client-only option")
		}
	}
	if initCmd.crdOnly && initCmd.wait {
		return errors.New("wait is not allowed with crd-only")
	}
	if flags.Changed("wait-timeout") && !initCmd.wait {
		return errors.New("wait-timeout is only useful when using the flag '--wait'")
	}

	return nil
}

// run initializes local config and installs KUDO manager to Kubernetes cluster.
func (initCmd *initCmd) run() error {
	opts := kudoinit.NewOptions(initCmd.version, initCmd.ns, initCmd.serviceAccount, initCmd.selfSignedWebhookCA)
	// if image provided switch to it.
	if initCmd.image != "" {
		opts.Image = initCmd.image
	}
	if initCmd.imagePullPolicy != "" {
		switch initCmd.imagePullPolicy {
		case
			"Always",
			"Never",
			"IfNotPresent":
			opts.ImagePullPolicy = initCmd.imagePullPolicy
		default:
			return fmt.Errorf("Unknown image pull policy %s, must be one of 'Always', 'IfNotPresent' or 'Never'", initCmd.imagePullPolicy)
		}
	}

	installer := setup.NewInstaller(opts, initCmd.crdOnly)

	// initialize client
	if !initCmd.dryRun {
		if err := initCmd.initialize(); err != nil {
			return clog.Errorf("error initializing: %s", err)
		}
	}

	// if output is yaml | json, we only print the requested output style.
	if initCmd.output == "" {
		clog.Printf("$KUDO_HOME has been configured at %s", Settings.Home)
	}
	if initCmd.clientOnly {
		return nil
	}

	// initialize server
	clog.V(4).Printf("initializing server")
	if initCmd.client == nil {
		client, err := kube.GetKubeClient(Settings.KubeConfig)
		if err != nil {
			return clog.Errorf("could not get Kubernetes client: %s", err)
		}
		initCmd.client = client
	}
	ok, err := initCmd.preInstallVerify(installer)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("failed to verify installation requirements")
	}

	//TODO: implement output=yaml|json (define a type for output to constrain)
	//define an Encoder to replace YAMLWriter
	if strings.ToLower(initCmd.output) == "yaml" {
		manifests, err := installer.AsYamlManifests()
		if err != nil {
			return err
		}
		if err := initCmd.YAMLWriter(initCmd.out, manifests); err != nil {
			return err
		}
	}

	if !initCmd.dryRun {
		if err := installer.Install(initCmd.client); err != nil {
			return clog.Errorf("error installing: %s", err)
		}

		if initCmd.wait {
			clog.Printf("⌛Waiting for KUDO controller to be ready in your cluster...")
			err := setup.WatchKUDOUntilReady(initCmd.client.KubeClient, opts, initCmd.timeout)
			if err != nil {
				return errors.New("watch timed out, readiness uncertain")
			}
		}
	}

	return nil
}

// preInstallVerify runs the pre-installation verification and returns true if the installation can continue
func (initCmd *initCmd) preInstallVerify(v kudoinit.InstallVerifier) (bool, error) {
	result := verifier.NewResult()
	if err := v.PreInstallVerify(initCmd.client, &result); err != nil {
		return false, err
	}
	result.PrintWarnings(initCmd.errOut)
	if !result.IsValid() {
		result.PrintErrors(initCmd.errOut)
		return false, nil
	}
	return true, nil
}

// YAMLWriter writes yaml to writer.   Looked into using https://godoc.org/gopkg.in/yaml.v2#NewEncoder which
// looks like a better way, however the omitted JSON elements are encoded which results in a very verbose output.
//TODO: Write a Encoder util which uses the "sigs.k8s.io/yaml" library for marshalling
func (initCmd *initCmd) YAMLWriter(w io.Writer, manifests []string) error {
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

//func initialize(fs afero.Fs, settings env.Settings, out io.Writer) error {
func (initCmd *initCmd) initialize() error {

	if err := ensureDirectories(initCmd.fs, initCmd.home); err != nil {
		return err
	}

	return ensureRepositoryFile(initCmd.fs, initCmd.home)
}

func ensureRepositoryFile(fs afero.Fs, home kudohome.Home) error {
	exists, err := afero.Exists(fs, home.RepositoryFile())
	if err != nil {
		return err
	}
	if !exists {
		clog.V(1).Printf("Creating %s \n", home.RepositoryFile())
		r := repo.NewRepositories()
		if err := r.WriteFile(fs, home.RepositoryFile(), 0644); err != nil {
			return err
		}
	} else {
		clog.V(1).Printf("%v exists", home.RepositoryFile())
	}

	return nil
}

func ensureDirectories(fs afero.Fs, home kudohome.Home) error {
	dirs := []string{
		home.String(),
		home.Repository(),
	}
	for _, dir := range dirs {
		exists, err := afero.Exists(fs, dir)
		if err != nil {
			return err
		}
		if !exists {
			clog.V(1).Printf("creating %s \n", dir)
			if err := fs.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("could not create %s: %s", dir, err)
			}
		} else {
			clog.V(1).Printf("%v exists", dir)
		}
	}
	return nil
}
