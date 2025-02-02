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

		if resource.Type == "frontend"{
			uplocalCmd = fmt.Sprintf("cd %s && ", resource.Name)
			for _, resource := range utils.ManifestData.Resources {
				uplocalCmd += fmt.Sprintf("%sHOST=%s ", resource.Name, resource.LocalHost)
			}
			uplocalCmd += resource.UpLocal

		} else if resource.Type == "api" {
			// cmdString := fmt.Sprintf("cd %s && rm kubefs.env; touch kubefs.env", resource.Name)
			// for _, resource := range utils.ManifestData.Resources {
			// 	cmdString += fmt.Sprintf(" && echo %sHOST=%s >> kubefs.env", resource.Name, resource.LocalHost)
			// }
			// command := exec.Command("sh", "-c", cmdString)
			// command.Stdout = os.Stdout
			// command.Stderr = os.Stderr
			// err := command.Run()
			// if err != nil {
			// 	utils.PrintError(fmt.Sprintf("Error setting up kubefs.env: %v", err))
			// 	return
			// }
		}else{
			utils.PrintError("Cannot run a database resource")
			return
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
