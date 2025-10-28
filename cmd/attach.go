/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach [command]",
	Short: "kubefs attach - attach your current shell to inside a docker container or kubernetes pod",
	Long: `kubefs attach - attach your current shell to inside a docker container or kubernetes pod
example:
	kubefs attach <name> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		err, resource := utils.GetResourceFromName(args[0])
		if err != nil {
			utils.PrintError(err.Error())
			return
		}

		inKubernetes, _ := cmd.Flags().GetBool("attach-in-kubernetes")
		var command string
		if inKubernetes {
			command = resource.AttachCommand["kubernetes"]

			// get target
			target, _ := cmd.Flags().GetString("target")
			err, config := utils.VerifyCloudConfig(target)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error verifying target %s configuration", target))
			}

			// update context
			if target == "minikube" {
				err = utils.GetMinikubeContext(config)
			}else if target == "gcp" {
				err = utils.GetGCPClusterContext(config)
			}
			if err != nil {
				utils.PrintError(fmt.Sprintf("failed to switch to %s cluster context: %v", target, err))
			}
		}else{
			command = resource.AttachCommand["docker"]
		}

		utils.PrintWarning(fmt.Sprintf("Attaching to container %s. Use 'exit' or '\\q' to return", resource.Name))
		err = utils.RunCommand(command, true, true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error attaching to container: %v", err.Error()))
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.PersistentFlags().BoolP("attach-in-kubernetes", "k", false, "Attach to a kubernetes pod")
	attachCmd.PersistentFlags().StringP("target", "t", "minikube", "target environment to attach to ['minikube', 'gcp']")
}
