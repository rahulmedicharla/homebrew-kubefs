/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
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

func undeployAddon(addon *types.Addon) error {
	err := utils.RunCommand(fmt.Sprintf("helm uninstall %s", addon.Name), true, true)
	if err != nil {
		return err
	}

	return nil
}

func undeployUnique(resource *types.Resource) error {
	var cmd string
	if resource.Type == "database"{
		cmd = fmt.Sprintf("helm uninstall %s --namespace %s; kubectl delete namespace %s", resource.Name, resource.Name, resource.Name)
	}else{
		cmd = fmt.Sprintf("helm uninstall %s", resource.Name)
	}
	
	err := utils.RunCommand(cmd, true, true)
	if err != nil {
		return err
	}

	return nil
}

var undeployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs undeploy all - undeploy all the created resources from the clusters",
	Long: `kubefs undeploy all - undeploy all the created resources from the clusters
example:
	kubefs undeploy all --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}
		
		var closeCluster, pauseCluster bool
		closeCluster, _ = cmd.Flags().GetBool("close")
		pauseCluster, _ = cmd.Flags().GetBool("pause")

        utils.PrintWarning("Undeploying all resources")
		utils.PrintWarning("Undeploying all addons")

		var errors []string
		var successes []string

        for _, resource := range utils.ManifestData.Resources {
			err := undeployUnique(&resource)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s. %v", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}
			successes = append(successes, resource.Name)
        }

		for _, addon := range utils.ManifestData.Addons {
			err := undeployAddon(&addon)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error undeploying addon %s. %v", addon.Name, err.Error()))
				errors = append(errors, addon.Name)
				continue
			}
			successes = append(successes, addon.Name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error undeploying resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v undeployed successfully", successes))
		}

		if pauseCluster {
			utils.PrintWarning("Pausing the cluster")
			err := utils.RunCommand("minikube pause", true, true)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error pausing the cluster: %v", err))
				return
			}
		}

		if closeCluster {
			utils.PrintWarning("Closing the cluster")
			err := utils.RunCommand("minikube stop", true, true)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error closing the cluster: %v", err))
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

		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var closeCluster, pauseCluster bool
		closeCluster, _ = cmd.Flags().GetBool("close")
		pauseCluster, _ = cmd.Flags().GetBool("pause")

		names := strings.Split(args[0], ",")
		
		addons, _ := cmd.Flags().GetString("with-addons")
		var addonList []string
		if addons != "" {
			addonList = strings.Split(addons, ",")
		}

		var errors []string
		var successes []string

        utils.PrintWarning(fmt.Sprintf("Undeploying resource %v", names))
		utils.PrintWarning(fmt.Sprintf("Undeploying addons %v", addonList))

		for _, name := range names {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = undeployUnique(resource)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
			}
			successes = append(successes, name)
		}

		for _, name := range addonList {
			addon, err := utils.GetAddonFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = undeployAddon(addon)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error undeploying addon %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
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
			err := utils.RunCommand("minikube pause", true, true)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error pausing the cluster: %v", err))
				return
			}
		}

		if closeCluster {
			utils.PrintWarning("Closing the cluster")
			err := utils.RunCommand("minikube stop", true, true)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error closing the cluster: %v", err))
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

	undeployResourceCmd.Flags().StringP("with-addons", "a", "", "include addons in the undeploy")
}
