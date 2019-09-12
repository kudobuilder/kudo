package cmd

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const repoIndexDesc = `
Read the provided directory and generate an index file based on the packages found.

This tool is used for creating an 'index.yaml' file for a KUDO package repository. To
set an absolute URL to the operators, use '--url' flag.

# Create an index file for all KUDO packages in the repo-dir.
	$ kubectl kudo repo index repo-dir`

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
		Use:   "index [flags] <DIR>",
		Short: "Generate an index file given a directory containing KUDO packages",
		Long:  repoIndexDesc,
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
	f.StringVar(&index.url, "url", "", "URL of the operator repository")
	f.StringVar(&index.urlRepoName, "url-repo", "", "Name of the repo to use URL for index entry urls")
	f.StringVar(&index.mergePath, "merge", "", "URL or path location of index file to merge with")
	f.StringVar(&index.mergeRepoName, "merge-repo", "", "Name of the repo to use a merge URL")
	f.BoolVarP(&index.overwrite, "overwrite", "o", false, "Overwrite existing package")

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
		config, err := ri.repoConfig()
		if err != nil {
			return err
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

		// index is the master, any dups in the merged in index will have what is local replace those entries
		for _, pvs := range mergeIndex.Entries {
			for _, pv := range pvs {
				err := index.AddPackageVersion(pv)
				if err != nil {
					// todo: add verbose logging here
					continue
				}
			}
		}
	}

	index.WriteFile(ri.fs, target)
	fmt.Fprintf(ri.out, "index %v created.\n", target)
	return nil
}

func (ri *repoIndexCmd) mergeRepoConfig() (*repo.Configuration, error) {
	if ri.mergeRepoName != "" {
		return ri.repoConfig()
	}

	return &repo.Configuration{
		URL:  ri.mergePath,
		Name: "temp-merge-path",
	}, nil
}

func (ri *repoIndexCmd) repoConfig() (*repo.Configuration, error) {
	repos, err := repo.LoadRepositories(ri.fs, Settings.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}
	return repos.GetConfiguration(ri.mergeRepoName), nil
}
