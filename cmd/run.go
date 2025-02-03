/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"os"
	"os/exec"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "kubefs run - run a resource locally (dev)",
	Long: `kubefs run - run a resource locally (dev)`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		name := args[0]
		utils.PrintWarning(fmt.Sprintf("Running resource %s", name))

		var resource *types.Resource
		resource = utils.GetResourceFromName(name)
		var uplocalCmd string

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
			return
		}

		if resource.Type == "database"{
			utils.PrintError(fmt.Sprintf("Cannot run database resource %s", name))
			return
		}else {
			uplocalCmd = fmt.Sprintf("cd %s && ", resource.Name)
			for _, resource := range utils.ManifestData.Resources {
				uplocalCmd += fmt.Sprintf("%sHOST=%s ", resource.Name, resource.LocalHost)
			}
			uplocalCmd += resource.UpLocal
		}

		utils.PrintWarning(fmt.Sprintf("Running command %s", uplocalCmd))
		command := exec.Command("sh", "-c", uplocalCmd)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
