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
	"strings"
)

// undeployCmd represents the undeploy command
var undeployCmd = &cobra.Command{
	Use:   "undeploy [command]",
	Short: "kubefs undeploy - undeploy the created resources from the clusters",
	Long: `kubefs undeploy - undeploy the created resources from the clusters
example:
	kubefs undeploy all --flags,
	kubefs undeploy resource <frontend>,<api>,<database> --flags,
	kubefs undeploy resource <frontend> --flags`,
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
		var closeCluster, pauseCluster bool
		closeCluster, _ = cmd.Flags().GetBool("close")
		pauseCluster, _ = cmd.Flags().GetBool("pause")


		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Undeploying all resources")

		var errors []string
		var successes []string

        for _, resource := range utils.ManifestData.Resources {
			err := undeployUnique(&resource)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s", resource.Name))
				errors = append(errors, resource.Name)
				break
			}
			successes = append(successes, resource.Name)
        }

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error undeploying resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v undeployed successfully", successes))
		}

		if pauseCluster {
			utils.PrintWarning("Pausing the cluster")
			cmd := exec.Command("sh", "-c", "minikube pause")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error pausing the cluster: %v", cmdErr))
				return
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
	},
}

var undeployResourceCmd = &cobra.Command{
	Use:   "resource [name, ...]",
	Short: "kubefs undeploy resource - undeploy listed resource from the clusters",
	Long: `kubefs undeploy resource - undeploy listed resource from the clusters
example:
	kubefs undeploy resource <frontend>,<api>,<database> --flags,
	kubefs undeploy resource <frontend> --flags
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

		var closeCluster, pauseCluster bool
		closeCluster, _ = cmd.Flags().GetBool("close")
		pauseCluster, _ = cmd.Flags().GetBool("pause")
		names := strings.Split(args[0], ",")

		var errors []string
		var successes []string

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
				errors = append(errors, name)
				break
			}
			successes = append(successes, name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error undeploying resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v undeployed successfully", successes))
		}

		if pauseCluster {
			utils.PrintWarning("Pausing the cluster")
			cmd := exec.Command("sh", "-c", "minikube pause")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error pausing the cluster: %v", cmdErr))
				return
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
	},
}


func init() {
	rootCmd.AddCommand(undeployCmd)
	undeployCmd.AddCommand(undeployAllCmd)
	undeployCmd.AddCommand(undeployResourceCmd)

	undeployCmd.PersistentFlags().BoolP("close", "c", false, "Stop the cluster after undeploying the resources")
	undeployCmd.PersistentFlags().BoolP("pause", "p", false, "Pause the cluster after undeploying the resources")
}
