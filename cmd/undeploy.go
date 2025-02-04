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
	"os"
)

// undeployCmd represents the undeploy command
var undeployCmd = &cobra.Command{
	Use:   "undeploy [command]",
	Short: "kubefs undeploy - undeploy the created resources from the clusters",
	Long: `kubefs undeploy - undeploy the created resources from the clusters
example:
	kubefs undeploy all --flags,
	kubefs undeploy resource my-api my-frontend my-database --flags,
	kubefs undeploy resource my-api --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func undeployUnique(resource *types.Resource) int {
	var cmd *exec.Cmd
	if resource.Type == "database"{
		cmd = exec.Command("sh", "-c", fmt.Sprintf("helm uninstall %s --namespace %s; kubectl delete namespace %s", resource.Name, resource.Name, resource.Name))
	}else{
		cmd = exec.Command("sh", "-c", fmt.Sprintf("helm uninstall %s", resource.Name))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdErr := cmd.Run()
	if cmdErr != nil {
		utils.PrintError(fmt.Sprintf("Error undeploying resource %s: %v", resource.Name, cmdErr))
		return types.ERROR
	}

	return types.SUCCESS

}

var undeployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs undeploy all - undeploy all the created resources from the clusters",
	Long: `kubefs undeploy all - undeploy all the created resources from the clusters
example:
	kubefs undeploy all --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		var closeCluster bool
		closeCluster, _ = cmd.Flags().GetBool("close")

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Undeploying all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := undeployUnique(&resource)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s", resource.Name))
			}
        }

		if closeCluster {
			utils.PrintWarning("Closing the cluster")
			cmd := exec.Command("sh", "-c", "minikube stop")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error closing the cluster: %v", cmdErr))
				return
			}
		}

        utils.PrintSuccess("All resources undeployed successfully")
	},
}

var undeployResourceCmd = &cobra.Command{
	Use:   "resource [name, ...]",
	Short: "kubefs undeploy resource - undeploy listed resource from the clusters",
	Long: `kubefs undeploy resource - undeploy listed resource from the clusters
example:
	kubefs undeploy resource my-api my-frontend my-database --flags,
	kubefs undeploy resource my-api --flags
	`,
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
		closeCluster, _ = cmd.Flags().GetBool("close")
		names := args

        utils.PrintWarning(fmt.Sprintf("Undeploying resource %v", names))

		for _, name := range names {
			
			var resource *types.Resource
			resource = utils.GetResourceFromName(name)

			if resource == nil {
				utils.PrintError(fmt.Sprintf("Resource %s not found", name))
				break
			}

			err := undeployUnique(resource)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s", name))
				break
			}

		}

		if closeCluster {
			utils.PrintWarning("Closing the cluster")
			cmd := exec.Command("sh", "-c", "minikube stop")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error closing the cluster: %v", cmdErr))
				return
			}
		}

        utils.PrintSuccess(fmt.Sprintf("Resource %v undeployed successfully", names))
	},
}


func init() {
	rootCmd.AddCommand(undeployCmd)
	undeployCmd.AddCommand(undeployAllCmd)
	undeployCmd.AddCommand(undeployResourceCmd)

	undeployCmd.PersistentFlags().BoolP("close", "c", false, "Stop the cluster after undeploying the resources")
}
