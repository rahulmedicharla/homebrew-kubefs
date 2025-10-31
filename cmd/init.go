/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "kubefs init - initialize a new kubefs project",
	Long: `kubefs init - initialize a new kubefs project
example:
	kubefs init my-project
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		projectName := args[0]
		var description string

		err := utils.ReadInput("Enter project description: ", &description)
		if err != nil {
			utils.PrintError(fmt.Errorf("error reading project description: %v", err))
			return
		}

		commands := []string{
			fmt.Sprintf("mkdir %s", projectName),
			fmt.Sprintf("mkdir %s/addons", projectName),
		}

		err = utils.RunMultipleCommands(commands, false, true)
		if err != nil {
			utils.PrintError(fmt.Errorf("couldn't initialize project: %v", err))
		}

		cloudConfig := types.CloudConfig{
			ClusterNames: make([]string, 0),
		}

		project := types.Project{
			KubefsName:  projectName,
			Version:     "0.0.1",
			Description: description,
			Resources:   map[string]types.Resource{},
			Addons:      map[string]types.Addon{},
			CloudConfig: map[string]types.CloudConfig{
				"minikube": cloudConfig,
			},
		}

		err = utils.WriteManifest(&project, fmt.Sprintf("%s/manifest.yaml", projectName))
		if err != nil {
			fmt.Printf("Error writing manifest: %v\n", err)
			return
		}

		utils.PrintInfo(fmt.Sprintf("Project %s initialized successfully", projectName))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
