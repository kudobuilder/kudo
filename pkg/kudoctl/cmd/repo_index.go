package cmd

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const (
	repoIndexDesc = `
Read the provided directory and generate an index file based on the packages found.

This tool is used for creating an 'index.yaml' file for a KUDO package repository. To
set an absolute URL to the operators, use '--url' or '--url-repo' flag. The '--url-repo'
will look up the the url of the named repo and will provide the absolute URL by repo name.

To merge the generated index with an existing index file, use the '--merge' flag. In this 
case, the operator packages found in the current directory will be merged into the existing
index, with local operators taking priority over existing operators. No content of the existing
index file is modified. The use of '--url' or 'url-repo' will only apply to local operator
definitions in the index file. When using '--merge' it is necessary to specify the URL of the
index file to merge. The '--merge-repo' is the same as merge where the url is looked up by
repo name out of the local repositories list.

# Create an index file for all KUDO packages in the repo-dir.
	$ kubectl kudo repo index repo-dir`

	repoIndexExample = `  #simple index of operators in /opt/repo
  kubectl kudo repo index /opt/repo
  kubectl kudo repo index /opt/repo --url https://kudo-repository.storage.googleapis.com
	
  # merge with community repo
  kubectl kudo repo index /opt/repo --merge https://kudo-repository.storage.googleapis.com
  kubectl kudo repo index /opt/repo --merge-repo community	
`
)

type repoIndexCmd struct {
	path          string
	url           string
	urlRepoName   string
	overwrite     bool
	mergeRepoName string
	mergePath     string
	out           io.Writer
	time          *time.Time
	fs            afero.Fs
}

// newRepoIndexCmd for repo commands such as building a repo index file.
func newRepoIndexCmd(fs afero.Fs, out io.Writer, time *time.Time) *cobra.Command {
	index := &repoIndexCmd{out: out, fs: fs, time: time}
	cmd := &cobra.Command{
		Use:     "index [flags] <DIR>",
		Short:   "Generate an index file given a directory containing KUDO operator packages",
		Long:    repoIndexDesc,
		Example: repoIndexExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := index.validate(args); err != nil {
				return err
			}
			index.path = args[0]
			if err := index.run(); err != nil {
				return err
			}
			return nil
		},
		SilenceUsage: true,
	}

	f := cmd.Flags()
	f.StringVar(&index.url, "url", "", "URL of the operators to reference in the index file")
	f.StringVar(&index.urlRepoName, "url-repo", "", "Name of the repo to use URL for operator urls")
	f.StringVar(&index.mergePath, "merge", "", "URL or path location of index file to merge with")
	f.StringVar(&index.mergeRepoName, "merge-repo", "", "Name of the repo to use as merge URL")
	f.BoolVarP(&index.overwrite, "overwrite", "w", false, "Overwrite existing package")

	return cmd
}

func (ri *repoIndexCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - directory containing the operators to package")
	}
	// we do not allow the setting of merge and merge-repo
	if ri.mergeRepoName != "" && ri.mergePath != "" {
		return errors.New("specify either 'merge' or 'merge-repo', not both")
	}
	// we do not allow the setting of merge and merge-repo
	if ri.url != "" && ri.urlRepoName != "" {
		return errors.New("specify either 'url' or 'url-repo', not both")
	}

	return nil
}

// run returns the errors associated with cmd env
func (ri *repoIndexCmd) run() error {

	if ri.urlRepoName != "" {
		config, err := ri.repoConfig(ri.urlRepoName)
		if err != nil {
			return err
		}
		if config == nil {
			return fmt.Errorf("configuration for repositories does not contain %s", ri.urlRepoName)
		}
		ri.url = config.URL
	}

	target, err := files.FullPathToTarget(ri.fs, ri.path, "index.yaml", ri.overwrite)
	if err != nil {
		return err
	}

	index, err := repo.IndexDirectory(ri.fs, ri.path, ri.url, ri.time)
	if err != nil {
		return err
	}
	// if we have a merge path... lets get it
	if ri.mergePath != "" || ri.mergeRepoName != "" {

		config, err := ri.mergeRepoConfig()
		if err != nil {
			return err
		}
		client, err := repo.NewClient(config)
		if err != nil {
			return err
		}
		// valid the url and that we can pull and index is valid
		mergeIndex, err := client.DownloadIndexFile()
		if err != nil {
			return err
		}
		merge(index, mergeIndex)
	}

	if err := index.WriteFile(ri.fs, target); err != nil {
		return err
	}
	fmt.Fprintf(ri.out, "index %v created.\n", target)
	return nil
}

func merge(index *repo.IndexFile, mergeIndex *repo.IndexFile) {
	// index is the reference copy which will have non-duplicated entries added to it from the mergedIndex
	for _, pvs := range mergeIndex.Entries {
		for _, pv := range pvs {
			err := index.AddPackageVersion(pv)
			// this is most likely to be a duplicate pv, which we ignore (but will log at the right v)
			if err != nil {
				// todo: add verbose logging here
				continue
			}
		}
	}
}

func (ri *repoIndexCmd) mergeRepoConfig() (*repo.Configuration, error) {
	if ri.mergeRepoName != "" {
		return ri.repoConfig(ri.mergeRepoName)
	}

	return &repo.Configuration{
		URL:  ri.mergePath,
		Name: "temp-merge-path",
	}, nil
}

func (ri *repoIndexCmd) repoConfig(repoName string) (*repo.Configuration, error) {
	repos, err := repo.LoadRepositories(ri.fs, Settings.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}
	return repos.GetConfiguration(repoName), nil
}
