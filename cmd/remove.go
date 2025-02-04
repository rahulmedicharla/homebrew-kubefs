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
	"github.com/zalando/go-keyring"
	"strings"
	"os"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [command]",
	Short: "kubefs remove - delete listed resource locally and from docker hub",
	Long: `kubefs remove - delete listed resource locally and from docker hub
example:
	kubefs remove all --flags,
	kubefs remove resource my-api my-frontend my-database --flags,
	kubefs remove resource my-api --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func removeUnique(resource *types.Resource, onlyLocal bool, onlyRemote bool) int {
	if !onlyRemote {
		// remove locally
		cmd := exec.Command("sh", "-c", fmt.Sprintf("rm -rf %s", resource.Name))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error removing resource: %v", err))
			return types.ERROR
		}

		manifestErr := utils.RemoveResource(&utils.ManifestData, resource.Name)
		if manifestErr == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error removing resource: %v", manifestErr))	
			return types.ERROR
		}
	}

	if !onlyLocal && resource.Type != "database" {
		// remove from docker hub
		url := "https://hub.docker.com/v2/users/login/"

		headers := map[string]string{
			"Content-Type": "application/json",
		}

		creds, err := keyring.Get("docker", "kubefs")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error getting Docker credentials: %v", err))
			return types.ERROR 
		}

		username, pat := strings.Split(creds, ":")[0], strings.Split(creds, ":")[1]

		payload := map[string]interface{}{
			"username": username,
			"password": pat,
		}

		status, response, err := utils.PostRequest(url, headers, payload)
		if status == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error logging into Docker: %v", err))
			return types.ERROR 
		}

		if response.Token == "" {
			utils.PrintError(fmt.Sprintf("Error logging into Docker: No token received. %s", response.Detail))
			return types.ERROR 
		}

		url = fmt.Sprintf("https://hub.docker.com/v2/repositories/%s", resource.DockerRepo)
		headers = map[string]string{
			"Authorization": fmt.Sprintf("JWT %s", response.Token),
		}

		status, err = utils.DeleteRequest(url, headers)
		if status == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error deleting resource from Docker: %v", err))
			return types.ERROR
		}
	}

	return types.SUCCESS

}

var removeAllCmd = &cobra.Command{
    Use:   "all",
    Short: "kubefs remove all - remove all resources locally and from docker hub",
    Long:  `kubefs remove all - remove all resources locally and from docker hub
example:
	kubefs remove all --flags
	`,
    Run: func(cmd *cobra.Command, args []string) {
		var onlyLocal, onlyRemote bool
		onlyLocal, _ = cmd.Flags().GetBool("only-local")
		onlyRemote, _ = cmd.Flags().GetBool("only-remote")

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Removing all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := removeUnique(&resource, onlyLocal, onlyRemote)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error removing resource %s", resource.Name))
			}
        }

        utils.RemoveAll(&utils.ManifestData)
        utils.PrintSuccess("All resources removed successfully")
    },
}

var removeResourceCmd = &cobra.Command{
    Use:   "resource [name, ...]",
    Short: "kubefs remove resource [name, ...] - remove listed resource locally and from docker hub",
    Long:  `kubefs remove resource [name, ...] - remove listed resource locally and from docker hub
example:
	kubefs remove resource my-api my-frontend my-database --flags,
	kubefs remove resource my-api --flags
`,
    Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		names := args 

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var onlyLocal, onlyRemote bool
		onlyLocal, _ = cmd.Flags().GetBool("only-local")
		onlyRemote, _ = cmd.Flags().GetBool("only-remote")		

        utils.PrintWarning(fmt.Sprintf("Removing resource %v", names))

		for _, name := range names {
			var resource *types.Resource
			resource = utils.GetResourceFromName(name)

			if resource == nil {
				utils.PrintError(fmt.Sprintf("Resource %s not found", name))
				break
			}

			err := removeUnique(resource, onlyLocal, onlyRemote)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error removing resource %s", name))
				break
			}
		}

		utils.PrintSuccess(fmt.Sprintf("Resource %v removed successfully", names))

    },
}


func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeAllCmd)
	removeCmd.AddCommand(removeResourceCmd)

	removeCmd.PersistentFlags().BoolP("only-local", "l", false, "only remove the resource locally")
	removeCmd.PersistentFlags().BoolP("only-remote", "r", false, "only remove the resource from docker hub")
}
