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
	Short: "kubefs remove - delete a resource locally and from docker hub",
	Long: "kubefs remove - delete a resource locally and from docker hub",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func removeUnique(name string, onlyLocal bool, onlyRemote bool, docker_repo string, resource_type string, framework string) int {
	if !onlyRemote {
		// remove locally
		commands := []string{
			fmt.Sprintf("cd %s; docker compose down -v; docker rmi %s:latest; docker network rm shared_network; echo ''", name, docker_repo),
			fmt.Sprintf("rm -rf %s", name),
		}

		if resource_type == "frontend"{
			commands = append(commands, fmt.Sprintf("docker rmi traefik:latest; echo ''"))
		}else if resource_type == "database"{
			if framework == "mongodb" {
				commands = append(commands, fmt.Sprintf("docker rmi mongo:latest; echo ''"))
			}else{
				commands = append(commands, fmt.Sprintf("docker rmi cassandra:latest; echo ''"))
			}
			commands = append(commands, fmt.Sprintf("docker volume prune; echo ''", name) )
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error removing resource: %v", err))
				return types.ERROR
			}
		}

		manifestErr := utils.RemoveResource(&utils.ManifestData, name)
		if manifestErr == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error removing resource: %v", manifestErr))	
			return types.ERROR
		}
	}

	if !onlyLocal && docker_repo != "" {
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

		url = fmt.Sprintf("https://hub.docker.com/v2/repositories/%s", docker_repo)
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
    Long:  "kubefs remove all - remove all resources locally and from docker hub",
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
			err := removeUnique(resource.Name, onlyLocal, onlyRemote, resource.DockerRepo, resource.Type, resource.Framework)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error removing resource %s", resource.Name))
			}
        }

        utils.RemoveAll(&utils.ManifestData)
        utils.PrintSuccess("All resources removed successfully")
    },
}

var removeResourceCmd = &cobra.Command{
    Use:   "resource [name]",
    Short: "kubefs remove resource [name] - remove a specific resource locally and from docker hub",
    Long:  "kubefs remove resource [name] - remove a specific resource locally and from docker hub",
    Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var onlyLocal, onlyRemote bool
		onlyLocal, _ = cmd.Flags().GetBool("only-local")
		onlyRemote, _ = cmd.Flags().GetBool("only-remote")		

        name := args[0]
        utils.PrintWarning(fmt.Sprintf("Removing resource %s", name))

		var dockerRepo string
		var resourceType string
		var resourceFramework string
		for _, resource := range utils.ManifestData.Resources {
			if resource.Name == name {
				dockerRepo = resource.DockerRepo
				resourceType = resource.Type
				resourceFramework = resource.Framework
				break
			}
		}

		err := removeUnique(name, onlyLocal, onlyRemote, dockerRepo, resourceType, resourceFramework)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error removing resource %s", name))
			return
		}

        utils.PrintSuccess(fmt.Sprintf("Resource %s removed successfully", name))
    },
}


func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeAllCmd)
	removeCmd.AddCommand(removeResourceCmd)

	removeCmd.PersistentFlags().BoolP("only-local", "l", false, "only remove the resource locally")
	removeCmd.PersistentFlags().BoolP("only-remote", "r", false, "only remove the resource from docker hub")
}
