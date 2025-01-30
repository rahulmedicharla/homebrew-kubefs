/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"reflect"
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   "describe [command]",
	Short: "kubefs describe - describe a resource",
	Long: "kubefs describe - describe a resource ",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var describeAllCmd = &cobra.Command{
    Use:   "all",
    Short: "kubefs describe all - describe all resources",
    Long:  "kubefs describe all - describe all resources",
    Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		utils.PrintWarning("Describing all resources")

		for _, resource := range utils.ManifestData.Resources {
			resourceValue := reflect.ValueOf(resource)
			resourceType := reflect.TypeOf(resource)
			for i := 0; i < resourceValue.NumField(); i++ {
				field := resourceType.Field(i)
				value := resourceValue.Field(i)
				fmt.Printf("%s: %v\n", field.Name, value)
			}
			fmt.Println("\n")
		}
    },
}

var describeResourceCmd = &cobra.Command{
    Use:   "resource [name]",
    Short: "kubefs describe resource [name] - describe a specific resource",
    Long:  "kubefs describe resource [name] - describe a specific resource",
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
		utils.PrintWarning(fmt.Sprintf("Describing resource %s\n", name))

		for _, resource := range utils.ManifestData.Resources {
			if resource.Name == name {
				resourceValue := reflect.ValueOf(resource)
				resourceType := reflect.TypeOf(resource)
				for i := 0; i < resourceValue.NumField(); i++ {
					field := resourceType.Field(i)
					value := resourceValue.Field(i)
					fmt.Printf("%s: %v\n", field.Name, value)
				}
				return
			}
		}

		utils.PrintError(fmt.Sprintf("Resource %s not found", name))
		return
    },
}


func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.AddCommand(describeAllCmd)
	describeCmd.AddCommand(describeResourceCmd)
}
