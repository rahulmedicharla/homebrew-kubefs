/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"

)

// undeployCmd represents the undeploy command
var undeployCmd = &cobra.Command{
	Use:   "undeploy [command]",
	Short: "kubefs undeploy - undeploy the created resources from the clusters",
	Long: `kubefs undeploy - undeploy the created resources from the clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func undeployUnique(name string, closeCluster bool, target string) int {
	// do something

	return types.SUCCESS

}

var undeployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs undeploy all - undeploy all the created resources from the clusters",
	Long: `kubefs undeploy all - undeploy all the created resources from the clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		var closeCluster bool
		var target string
		closeCluster, _ = cmd.Flags().GetBool("close")
		target, _ = cmd.Flags().GetString("target")

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		if target != "local" && target != "aws" && target != "gcp" && target != "azure" {
			utils.PrintError("Invalid target cluster: use 'local', 'aws', 'gcp', or 'azure'")
			return
		}

        utils.PrintWarning("Undeploying all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := undeployUnique(resource.Name, closeCluster, target)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s", resource.Name))
			}
        }

        utils.PrintSuccess("All resources undeployed successfully")
	},
}

var undeployResourceCmd = &cobra.Command{
	Use:   "resource [name]",
	Short: "kubefs undeploy resource - undeploy a specific resource from the clusters",
	Long: `kubefs undeploy resource - undeploy a specific resource from the clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var closeCluster bool
		var target string
		closeCluster, _ = cmd.Flags().GetBool("close")
		target, _ = cmd.Flags().GetString("target")

		if target != "local" && target != "aws" && target != "gcp" && target != "azure" {
			utils.PrintError("Invalid target cluster: use 'local', 'aws', 'gcp', or 'azure'")
			return
		}

        name := args[0]
        utils.PrintWarning(fmt.Sprintf("Undeploying resource %s", name))

		err := undeployUnique(name, closeCluster, target)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error undeploying resource %s", name))
			return
		}

        utils.PrintSuccess(fmt.Sprintf("Resource %s undeployed successfully", name))
	},
}


func init() {
	rootCmd.AddCommand(undeployCmd)
	undeployCmd.AddCommand(undeployAllCmd)
	undeployCmd.AddCommand(undeployResourceCmd)

	undeployCmd.PersistentFlags().StringP("target", "t", "local", "target cluster to undeploy the resources from [local|aws|gcp|azure]")
	undeployCmd.PersistentFlags().BoolP("close", "c", false, "Stop the cluster after undeploying the resources")
}
