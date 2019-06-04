// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	importcmd "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/import"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

// NewImportCmd creates the import command for the CLI
func NewImportCmd() *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import folder as Framework and FrameworkVersion",
		Long: `Imports a folder with the KUDO or Helm folder structure to be applied	
	

	kubectl kudo import -f /path/to/definition | kubectl apply -f -
	`,
		RunE: importcmd.Run,
	}

	importCmd.Flags().StringVarP(&vars.FrameworkImportPath, "folder", "f", "", "Folder directory to import")
	importCmd.Flags().StringVarP(&vars.Format, "out", "o", "json", "Output format")
	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	importCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(importCmd.OutOrStderr(), usageFmt, importCmd.UseLine(), importCmd.Flags().FlagUsages())
		return nil
	})

	return importCmd
}
