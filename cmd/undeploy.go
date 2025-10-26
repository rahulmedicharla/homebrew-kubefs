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
	kubefs undeploy resource <frontend> <api> <database> --flags,
	kubefs undeploy addons <addon-name> <addon-name> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func closeOrPauseCluster(closeCluster bool, pauseCluster bool, target string) error {
	if target == "local" {
		if closeCluster {
			err := utils.RunCommand("minikube stop", true, true)
			if err != nil {
				return fmt.Errorf("failed to stop local cluster: %v", err)
			}
		}
		if pauseCluster {
			err := utils.RunCommand("minikube pause", true, true)
			if err != nil {
				return fmt.Errorf("failed to pause local cluster: %v", err)
			}
		}
	} else if target == "gcp" {
		return fmt.Errorf("close and pause operations are not supported for GCP autopilot clusters. Please manually delet the cluster")
	}
	return nil
}

func undeployFromTarget(target string, commands []string) error {
	if target == "local" {
		// update context
		err := utils.RunCommand("kubectl config use-context minikube", true, true)
		if err != nil {
			return fmt.Errorf("failed to switch to local cluster context: %v", err)
		}

		// run commands
		return utils.RunMultipleCommands(commands, true, true)
	}else if target == "gcp" {
		// update context
		err, gcpConfig := utils.VerifyGcpProject()
		if err != nil {
			return err
		}

		// get kubeconfig for cluster
		err = utils.RunCommand(fmt.Sprintf("gcloud container clusters get-credentials %s --location %s", gcpConfig.ClusterName, gcpConfig.Region), true, true)
		if err != nil {
			return err
		}

		// deploy specified commands to GCP cluster
		err = utils.RunMultipleCommands(commands, true, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func undeployAddon(addon *types.Addon, target string) error {
	commands := []string{}
	if addon.Name == "oauth2"{
		commands = append(commands, "helm uninstall auth-data")
	}

	commands = append(commands, fmt.Sprintf("helm uninstall %s", addon.Name))

	err := undeployFromTarget(target, commands)
	if err != nil {
		return err
	}

	return nil
}

func undeployUnique(resource *types.Resource, target string) error {
	commandBuilder := strings.Builder{}
	commandBuilder.WriteString(fmt.Sprintf("helm uninstall %s;", resource.Name))
	
	if resource.Type == "database"{
		commandBuilder.WriteString(fmt.Sprintf("kubectl delete namespace %s;", resource.Name))
	}

	commands := []string{
		commandBuilder.String(),
	}

	err := undeployFromTarget(target, commands)
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
		target, _ := cmd.Flags().GetString("target")

		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

        utils.PrintWarning("Undeploying all resources")
		utils.PrintWarning("Undeploying all addons")

		var errors []string
		var successes []string

        for _, resource := range utils.ManifestData.Resources {
			err := undeployUnique(&resource, target)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error undeploying resource %s. %v", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}
			successes = append(successes, resource.Name)
        }

		for _, addon := range utils.ManifestData.Addons {
			err := undeployAddon(&addon, target)
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

		err = closeOrPauseCluster(closeCluster, pauseCluster, target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
	},
}

var undeployResourceCmd = &cobra.Command{
	Use:   "resource [name, ...]",
	Short: "kubefs undeploy resource - undeploy listed resource from the clusters",
	Long: `kubefs undeploy resource - undeploy listed resource from the clusters
example:
	kubefs undeploy resource <frontend> <api> <database> --flags,
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
		target, _ := cmd.Flags().GetString("target")

		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
		
		addons, _ := cmd.Flags().GetString("with-addons")
		var addonList []string
		if addons != "" {
			addonList = strings.Split(addons, ",")
		}

		var errors []string
		var successes []string

        utils.PrintWarning(fmt.Sprintf("Undeploying resource %v", args))
		utils.PrintWarning(fmt.Sprintf("Undeploying addons %v", addonList))

		for _, name := range args {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = undeployUnique(resource, target)
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

			err = undeployAddon(addon, target)
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

		err = closeOrPauseCluster(closeCluster, pauseCluster, target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
	},
}

var undeployAddonCmd = &cobra.Command{
	Use:   "addons [name, ...]",
	Short: "kubefs undeploy addon - undeploy listed addons from the clusters",
	Long: `kubefs undeploy addon - undeploy listed addons from the clusters
example:
	kubefs undeploy addon <addon-name> <addon-name> --flags,
	kubefs undeploy addon <addon-name> --flags
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
		target, _ := cmd.Flags().GetString("target")

		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Undeploying addons %v", args))

		for _, name := range args {
			addon, err := utils.GetAddonFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = undeployAddon(addon, target)
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

		err = closeOrPauseCluster(closeCluster, pauseCluster, target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
	},
}


func init() {
	rootCmd.AddCommand(undeployCmd)
	undeployCmd.AddCommand(undeployAllCmd)
	undeployCmd.AddCommand(undeployResourceCmd)
	undeployCmd.AddCommand(undeployAddonCmd)

	undeployCmd.PersistentFlags().StringP("target", "t", "local", "target cluster to undeploy the resources from ['local', 'gcp']")

	undeployCmd.PersistentFlags().BoolP("close", "c", false, "Stop the cluster after undeploying the resources")
	undeployCmd.PersistentFlags().BoolP("pause", "p", false, "Pause the cluster after undeploying the resources")

	undeployResourceCmd.Flags().StringP("with-addons", "a", "", "include addons in the undeploy")
}
