/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster [command]",
	Short: "kubefs cluster - manage clusters from provider",
	Long: `kubefs cluster - manage clusters from provider
example:
	kubefs cluster delete --flags
	kubefs cluster provision --flags
	kubefs cluster pause --flags
	kubefs cluster start --flags
	kubefs cluster main --flags
	kubefs cluster list --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var listClusterCmd = &cobra.Command{
	Use: "list",
	Short: "list the availbale clusters for a target to deploy on",
	Long: `list the available clusters for a target to deploy on
example: 
	kubefs cluster list --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		fmt.Printf("Target %s \n", config.Provider)
		fmt.Printf("\t Main Cluster: %s \n", config.MainCluster)
		for i,name := range config.ClusterNames {
			fmt.Printf("\t Cluster %v: %s \n", i, name)
		}

	},
}

var mainCmd = &cobra.Command{
	Use: "main",
	Short: "set the main cluster for a target to deploy on",
	Long: `set the main cluster for a target to deploy on
example: 
	kubefs cluster pause [clusterName] --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		// verify cloud config cluster and param matches
		clusterName := args[0]
		if !utils.VerifyClusterName(config, clusterName) {
			utils.PrintError(fmt.Sprintf("Cluster %s doesn't exist in target %s, please choose a different name or use 'kubefs cluster provision' ", clusterName, target))
			return 
		}

		utils.PrintSuccess(fmt.Sprintf("Setting target %s main cluster to %s", target, clusterName))

		config.MainCluster = clusterName
		err = utils.UpdateCloudConfig(&utils.ManifestData, target, config)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		utils.PrintSuccess("Main cluster configured successfully!")
		
	},
}


var pauseCmd = &cobra.Command{
	Use: "pause",
	Short: "pause a cluster",
	Long: `pause a cluster
example: 
	kubefs cluster pause [clusterName] --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		// verify cloud config cluster and param matches
		clusterName := args[0]
		if !utils.VerifyClusterName(config, clusterName) {
			utils.PrintError(fmt.Sprintf("Cluster %s doesn't exist, please choose a different name or use 'kubefs cluster provision' ", clusterName))
			return 
		}

		utils.PrintSuccess(fmt.Sprintf("Pausing cluster %s in target %s", clusterName, target))

		if target == "minikube"{
			// pause cluster
			err = utils.PauseMinikubeCluster(config, clusterName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}

		}else if target == "gcp"{
			utils.PrintWarning("gcp autopilot clusters don't support pausing/stopping")
			return
		}

	},
}

var startCmd = &cobra.Command{
	Use: "start",
	Short: "start a cluster",
	Long: `start a cluster
example: 
	kubefs cluster start [clusterName] --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		clusterName := args[0]

		// validate cluster exists
		if !utils.VerifyClusterName(config, clusterName) {
			utils.PrintError(fmt.Sprintf("Cluster %s doesn't exist, please choose a different name or use 'kubefs cluster provision' ", clusterName))
			return 
		}

		utils.PrintSuccess(fmt.Sprintf("Starting cluster %s in target %s", clusterName, target))

		if target == "minikube"{
			// start cluster
			err = utils.StartMinikubeCluster(config, clusterName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}

		}else if target == "gcp"{
			utils.PrintWarning("gcp autopilot clusters don't support starting clusters")
			return
		}

	},
}

var deleteCmd = &cobra.Command{
	Use: "delete",
	Short: "delete a cluster",
	Long: `delete a cluster
example: 
	kubefs cluster delete [clusterName] --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		clusterName := args[0]
		// verify cluster exists
		if !utils.VerifyClusterName(config, clusterName) {
			utils.PrintError(fmt.Sprintf("Cluster %s doesn't exist, please choose a different name or use 'kubefs cluster provision' ", clusterName))
			return 
		}

		utils.PrintSuccess(fmt.Sprintf("Deleting cluster %s in target %s", clusterName, target))

		if target == "minikube"{
			// delete cluster
			err = utils.DeleteMinikubeCluster(config, clusterName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}

		}else if target == "gcp"{
			// delete gcp cluster
			err = utils.DeleteGCPCluster(config, clusterName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}
		}

		// update Manifest
		_, config.ClusterNames = utils.RemoveClusterName(config, clusterName)
		if len(config.ClusterNames) > 0 {
			config.MainCluster = config.ClusterNames[0]
		}else {
			config.MainCluster = ""
		}

		err = utils.UpdateCloudConfig(&utils.ManifestData, target, config)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

	},
}

var provisionCmd = &cobra.Command{
	Use: "provision",
	Short: "provision a cluster",
	Long: `provision a cluster
example: 
	kubefs cluster provision [clusterName] --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		// Verify cloud provider target
		target, _ := cmd.Flags().GetString("target")
		err := utils.VerifyTarget(target)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		//  Verify authentication with cloud provider
		err, config := utils.VerifyCloudConfig(target)
		if err != nil{
			utils.PrintError(err.Error())
			return
		}

		clusterName := args[0]
		
		// validate cluster doesn't already exist
		if utils.VerifyClusterName(config, clusterName) {
			utils.PrintError(fmt.Sprintf("Cluster %s already exists, please choose a different name", clusterName))
			return 
		}
		
		if target == "minikube"{
			// provision minikube cluster
			err = utils.ProvisionMinikubeCluster(clusterName)
			if err != nil{
				utils.PrintError(err.Error())
				return
			}

		}else if target == "gcp"{
			// provision gcp cluster
			err = utils.ProvisionGcpCluster(config, clusterName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}
		}

		// update manifest
		config.ClusterNames = append(config.ClusterNames, clusterName)
		config.MainCluster = config.ClusterNames[0]
		err = utils.UpdateCloudConfig(&utils.ManifestData, target, config)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)

	clusterCmd.AddCommand(provisionCmd)
	clusterCmd.AddCommand(pauseCmd)
	clusterCmd.AddCommand(startCmd)
	clusterCmd.AddCommand(deleteCmd)
	clusterCmd.AddCommand(mainCmd)
	clusterCmd.AddCommand(listClusterCmd)

	clusterCmd.PersistentFlags().StringP("target", "t", "minikube", "target environment to deploy to ['minikube', 'gcp']")

}
