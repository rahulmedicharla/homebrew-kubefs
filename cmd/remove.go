/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [command]",
	Short: "kubefs remove - delete listed resource locally and from docker hub",
	Long: `kubefs remove - delete listed resource locally and from docker hub
example:
	kubefs remove all --flags,
	kubefs remove resource <frontend> <api> <database> --flags,
	kubefs remove resource <frontend> --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func removeUnique(resource *types.Resource, onlyLocal bool, onlyRemote bool) error {
	if !onlyRemote {
		// remove locally
		err := utils.RunCommand(fmt.Sprintf("rm -rf %s", resource.Name), true, true)
		if err != nil {
			return err
		}

		err = utils.RunCommand(fmt.Sprintf("docker images | grep %s", resource.DockerRepo), true, true)
		if err == nil {
			err = utils.RunCommand(fmt.Sprintf("docker rmi %s:latest", resource.DockerRepo), true, true)
			if err != nil {
				return err
			}
		}

		err = utils.RemoveResource(&utils.ManifestData, resource.Name)
		if err != nil {
			return err
		}
	}

	if !onlyLocal && resource.Type != "database" {
		// remove from docker hub

		creds, err := keyring.Get("docker", "kubefs")
		if err != nil {
			return err
		}

		username, pat := strings.Split(creds, ":")[0], strings.Split(creds, ":")[1]

		response, err := utils.PostRequest(types.DOCKER_LOGIN_ENDPOINT,
			map[string]string{
				"Content-Type": "application/json",
			}, map[string]interface{}{
				"username": username,
				"password": pat,
			},
		)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s%s", types.DOCKER_REPO_ENDPOINT, resource.DockerRepo)

		err = utils.DeleteRequest(url, map[string]string{
			"Authorization": fmt.Sprintf("JWT %s", response.Token),
		})

		if err != nil {
			return err
		}
	}

	return nil

}

var removeAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs remove all - remove all resources locally and from docker hub",
	Long: `kubefs remove all - remove all resources locally and from docker hub
example:
	kubefs remove all --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyLocal, onlyRemote bool
		onlyLocal, _ = cmd.Flags().GetBool("only-local")
		onlyRemote, _ = cmd.Flags().GetBool("only-remote")

		utils.PrintWarning("Removing all resources")

		var errors []string
		var successes []string

		for _, resource := range utils.ManifestData.Resources {
			err := removeUnique(&resource, onlyLocal, onlyRemote)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error removing resource %s. %v ", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}
			utils.RemoveResource(&utils.ManifestData, resource.Name)
			successes = append(successes, resource.Name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error removing resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Resource %v removed successfully", successes))
		}

	},
}

var removeResourceCmd = &cobra.Command{
	Use:   "resource [name ...]",
	Short: "kubefs remove resource [name ...] - remove listed resource locally and from docker hub",
	Long: `kubefs remove resource [name ...] - remove listed resource locally and from docker hub
example:
	kubefs remove resource <frontend> <api> <database> --flags,
	kubefs remove resource <frontend> --flags
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

		var onlyLocal, onlyRemote bool
		onlyLocal, _ = cmd.Flags().GetBool("only-local")
		onlyRemote, _ = cmd.Flags().GetBool("only-remote")

		utils.PrintWarning(fmt.Sprintf("Removing resource %v", args))

		var errors []string
		var successes []string

		for _, name := range args {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = removeUnique(resource, onlyLocal, onlyRemote)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error removing resource %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
			}
			utils.RemoveResource(&utils.ManifestData, name)
			successes = append(successes, name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error removing resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Resource %v removed successfully", successes))
		}

	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeAllCmd)
	removeCmd.AddCommand(removeResourceCmd)

	removeCmd.PersistentFlags().BoolP("only-local", "l", false, "only remove the resource locally")
	removeCmd.PersistentFlags().BoolP("only-remote", "r", false, "only remove the resource from docker hub")
}
