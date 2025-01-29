/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"os/exec"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [command]",
	Short: "kubefs remove - delete a resource locally and from docker hub",
	Long: "kubefs remove - delete a resource locally and from docker hub",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var removeAllCmd = &cobra.Command{
    Use:   "all",
    Short: "kubefs remove all - remove all resources locally and from docker hub",
    Long:  "kubefs remove all - remove all resources locally and from docker hub",
    Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Removing all resources")

        var commands []string
        for _, resource := range utils.ManifestData.Resources {
            commands = append(commands, fmt.Sprintf("rm -rf %s", resource.Name))
        }

        for _, command := range commands {
            cmd := exec.Command("sh", "-c", command)
            err := cmd.Run()
            if err != nil {
                utils.PrintError(fmt.Sprintf("Error removing resource: %v", err))
                return
            }
        }

        utils.RemoveAll(&utils.ManifestData)
        utils.PrintSuccess("All resources removed successfully")
    },
}

var removeResourceCmd = &cobra.Command{
    Use:   "resource [name]",
    Short: "kubefs remove resource [name] - remove a specific resource locally and from docker hub",
    Long:  "kubefs remove resource [name] - remove a specific resource locally and from docker hub",
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
        utils.PrintWarning(fmt.Sprintf("Removing resource %s", name))

		err := utils.RemoveResource(&utils.ManifestData, name)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error removing resource: %v", err))	
			return
		}

		command := exec.Command("sh", "-c", fmt.Sprintf("rm -rf %s", name))
		commandErr := command.Run()
		if commandErr != nil {
			utils.PrintError(fmt.Sprintf("Error removing resource: %v", commandErr))
			return
		}

        utils.PrintSuccess(fmt.Sprintf("Resource %s removed successfully", name))
    },
}


func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeAllCmd)
	removeCmd.AddCommand(removeResourceCmd)
}
