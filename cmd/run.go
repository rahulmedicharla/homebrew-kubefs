/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"strings"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "kubefs run - run a resource locally (dev)",
	Long: `kubefs run - run a resource locally (dev)
example:
	kubefs run <resource-name> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		if len(args) < 1 {
			cmd.Help()
			return
		}

		name := args[0]
		utils.PrintWarning(fmt.Sprintf("Running resource %s", name))

		var resource *types.Resource
		resource, err := utils.GetResourceFromName(name)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		upLocalCmd := strings.Builder{}
		if resource.Type == "database"{
			utils.PrintError(fmt.Sprintf("Cannot run database resource %s", name))
			return
		}else {
			upLocalCmd.WriteString(fmt.Sprintf("cd %s && ", resource.Name))
			for _, resource := range utils.ManifestData.Resources {
				upLocalCmd.WriteString(fmt.Sprintf("%sHOST=%s ", resource.Name, resource.LocalHost))
			}

			for _, name := range resource.Dependents {
				addon, err := utils.GetAddonFromName(name)
				if err != nil {
					utils.PrintError(err.Error())
					return
				}
				upLocalCmd.WriteString(fmt.Sprintf("%sHOST=%s ", addon.Name, addon.LocalHost))
			}

			upLocalCmd.WriteString(resource.UpLocal)
		}

		utils.PrintWarning(fmt.Sprintf("Running command %s", upLocalCmd.String()))
		utils.PrintSuccess(fmt.Sprintf("Resource %s is running locally", name))

		utils.RunCommand(upLocalCmd.String(), true, true)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
