/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/spf13/cobra"
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

		name := args[0]

		resource, err := utils.GetResourceFromName(name)
		if err != nil {
			utils.PrintError(err)
			return
		}

		inKubernetes, _ := cmd.Flags().GetBool("attach-in-kubernetes")
		var command string
		if inKubernetes {
			command = resource.AttachCommand["kubernetes"]

			// get target
			target, _ := cmd.Flags().GetString("target")
			config, err := utils.GetCloudConfigFromProvider(target)
			if err != nil {
				utils.PrintError(fmt.Errorf("error verifying target [%s] configuration", target))
			}

			// update context
			switch target {
			case "minikube":
				err = utils.GetMinikubeContext(config)
			case "gcp":
				err = utils.GetGCPClusterContext(config)
			}
			if err != nil {
				utils.PrintError(fmt.Errorf("failed to switch to [%s] cluster context: %v", target, err))
			}
		} else {
			command = resource.AttachCommand["docker"]
		}

		utils.PrintWarning(fmt.Sprintf("Attaching to container %s. Use 'exit' or '\\q' to return", name))
		err = utils.RunCommand(command, true, true)
		if err != nil {
			utils.PrintError(fmt.Errorf("error attaching to container: %v", err))
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.PersistentFlags().BoolP("attach-in-kubernetes", "k", false, "Attach to a kubernetes pod")
	attachCmd.PersistentFlags().StringP("target", "t", "minikube", "target environment to attach to ['minikube', 'gcp']")
}
