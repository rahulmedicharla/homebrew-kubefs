/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [command]",
	Short: "kubefs delete - delete cluster from provider",
	Long: `kubefs delete - delete cluster from provider
example:
	kubefs delete <provider>
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var deleteGCP = &cobra.Command{
	Use:   "gcp",
	Short: "Delete GCP cluster",
	Long:  `kubefs delete gcp - delete kubernetes cluster from GCP`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		err, config := utils.VerifyCloudConfig("gcp")
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		utils.PrintWarning(fmt.Sprintf("Deleting GCP cluster %s...", config.ClusterName))

		err = utils.DeleteGCPCluster(config)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
	},
}

var deleteMinikube = &cobra.Command{
	Use:   "minikube",
	Short: "Delete Minikube cluster",
	Long:  `kubefs delete minikube - delete kubernetes cluster from Minikube`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		err, config := utils.VerifyCloudConfig("minikube")
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		utils.PrintWarning(fmt.Sprintf("Deleting Minikube cluster %s...", config.ClusterName))

		err = utils.DeleteMinikubeCluster(config)
		if err != nil {
			utils.PrintError(err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.AddCommand(deleteGCP)
	deleteCmd.AddCommand(deleteMinikube)
}
