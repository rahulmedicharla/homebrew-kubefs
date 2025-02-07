/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"gopkg.in/yaml.v3"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <name> <description>",
	Short: "kubefs init - initialize a new kubefs project",
	Long: `kubefs init - initialize a new kubefs project
example:
	kubefs init my-project "My project description"
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		if len(args) < 1 {
			utils.PrintError("Please provide a name & description for the project")
			cmd.Help()
			return
		}

		projectName := args[0]
		description, _ := cmd.Flags().GetString("description")

		err := os.Mkdir(projectName, 0755)
		if err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			return
		}

		err = os.Mkdir(projectName + "/addons", 0755)
		if err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			return
		}

		project := types.Project{
			KubefsName: projectName,
			Version: "0.0.1",
			Description: description,
			Resources: []types.Resource{},
			Addons: []types.Addon{},
		}
	
		data, err := yaml.Marshal(&project)
		if err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			return
		}
	
		err = os.WriteFile(projectName + "/manifest.yaml", data, 0644)
		if err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			return
		}
		
		fmt.Println("Project initialized successfully")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringP("description", "d", "", "Description of the project")
	initCmd.MarkFlagRequired("description")
}
