/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"os"
	"os/exec"
	"os/signal"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "kubefs run - run a resource locally (dev)",
	Long: `kubefs run - run a resource locally (dev)`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		name := args[0]
		utils.PrintWarning(fmt.Sprintf("Running resource %s", name))

		var resource *types.Resource
		resource = utils.GetResourceFromName(name)

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
			return
		}

		var withKubefsHelper bool
		withKubefsHelper, _ = cmd.Flags().GetBool("with-kubefsHelper")

		if resource.Type == "database"{
			utils.PrintError("Cannot run a database resource")
			return
		} else if resource.Type == "api" {
			cmdString := fmt.Sprintf("cd %s && rm kubefs.env; touch kubefs.env", resource.Name)
			for _, resource := range utils.ManifestData.Resources {
				cmdString += fmt.Sprintf(" && echo %sHOST=%s >> kubefs.env", resource.Name, resource.LocalHost)
			}
			command := exec.Command("sh", "-c", cmdString)
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err := command.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error setting up kubefs.env: %v", err))
				return
			}
		}

		if withKubefsHelper {
			// check if kubefs helper is already running in a container
			command := exec.Command("sh", "-c", "docker ps | grep kubefsHelper")
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err := command.Run()
			if err != nil {
				// start kubefsHelper
				var cmdString = "docker run -d -p 6000:6000 --name kubefsHelper"
				for _, resource := range utils.ManifestData.Resources {
					cmdString += fmt.Sprintf(" -e %sHOST=%s", resource.Name, resource.LocalHost)
				}
				cmdString += " rmedicharla/kubefshelper"

				fmt.Sprintf("Starting kubefsHelper: %s", cmdString)

				command = exec.Command("sh", "-c", cmdString)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				err = command.Run()
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error starting kubefsHelper: %v", err))
					return
				}
			}else{
				withKubefsHelper = false
				utils.PrintWarning("kubefsHelper backend service already running")
			}
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			if withKubefsHelper {
				utils.PrintWarning("Stopping kubefsHelper backend service")
				command := exec.Command("sh", "-c", "docker stop kubefsHelper && docker rm kubefsHelper")
				err := command.Run()
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error stopping kubefsHelper: %v", err))
				}
			}
			os.Exit(0)
		}()

		command := exec.Command("sh", "-c", resource.UpLocal)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolP("with-kubefsHelper", "w", false, "Test your environment with the kubefsHelper backend service running")
}
