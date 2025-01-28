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
	Long: `kubefs init - initialize a new kubefs project`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		if len(args) < 2 {
			utils.PrintError("Please provide a name and description for the project")
			cmd.Help()
			return
		}

		projectName := args[0]
		description := args[1]

		err := os.Mkdir(projectName, 0755)
		if err != nil {
			fmt.Printf("Error initializing project: %v\n", err)
			return
		}

		project := types.Project{
			KubefsName: projectName,
			Version: "0.0.1",
			Description: description,
			Resources: []types.Resource{},
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
}
