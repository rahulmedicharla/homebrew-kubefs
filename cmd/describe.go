/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"reflect"
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:   "describe [command]",
	Short: "kubefs describe - describe a resource",
	Long: `kubefs describe - describe a resource 
example: 
	kubefs describe all,
	kubefs describe resource <frontend> <api> <database>
	kubefs describe resource <frontend>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var describeAllCmd = &cobra.Command{
    Use:   "all",
    Short: "kubefs describe all - describe all resources",
    Long:  `kubefs describe all - describe all resources
example: 
	kubefs describe all
	`,
    Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
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
    Use:   "resource [name ...]",
    Short: "kubefs describe resource [name ...] - describe listed resource",
    Long:  `kubefs describe resource [name ...] - describe a specific resource
example: 
	kubefs describe resource <frontend> <api> <database>
	kubefs describe resource <frontend>
	`,
    Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		utils.PrintWarning(fmt.Sprintf("Describing resource %v\n", args))

		for _ , name := range args {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				continue
			}

			resourceValue := reflect.ValueOf((*resource))
			resourceType := reflect.TypeOf((*resource))
			for i := 0; i < resourceValue.NumField(); i++ {
				field := resourceType.Field(i)
				value := resourceValue.Field(i)
				fmt.Printf("%s: %v\n", field.Name, value)
			}
			fmt.Println("\n")
		}
    },
}


func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.AddCommand(describeAllCmd)
	describeCmd.AddCommand(describeResourceCmd)
}
