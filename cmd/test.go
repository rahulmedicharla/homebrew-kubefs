/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"os/exec"
	"os"
	"github.com/rahulmedicharla/kubefs/types"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test [command]",
	Short: "kubefs test - test your build environment in docker locally before deploying",
	Long: `kubefs test - test your build environment in docker locally before deploying
example:
	kubefs test all --flags,
	kubefs test resource my-api my-frontend my-database --flags,
	kubefs test resource my-api --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var rawCompose = map[string]interface{}{
	"services": map[string]interface{}{},
	"networks": map[string]interface{}{
		"shared_network": map[string]string{
			"driver": "bridge",
		},
	},
	"volumes": map[string]interface{}{},
}

func modifyRawCompose(rawCompose *map[string]interface{}, resource *types.Resource) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker pull %s", resource.DockerRepo))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error pulling docker image. Run 'kubefs compile' to set: %v", err))
		return
	}

	service := map[string]interface{}{
		"image": resource.DockerRepo,
		"networks": []string{
			"shared_network",
		},
		"environment": []string{},
	}

	if resource.Type == "api" || resource.Type == "frontend" {
		service["ports"] = []string{
			fmt.Sprintf("%v:%v", resource.Port, resource.Port),
		}

		for _, r := range utils.ManifestData.Resources {
			service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%sHOST=%s", r.Name, r.DockerHost))
		}

	}else{
		if resource.Framework == "redis" {
			service["environment"] = []string{fmt.Sprintf("REDIS_PASSWORD=%s", resource.DbPassword)}
			service["ports"] = []string{fmt.Sprintf("%v:%v", resource.Port, resource.Port)}
			service["command"] = []string{"redis-server", fmt.Sprintf("--port %v", resource.Port), fmt.Sprintf("--requirepass %s", resource.DbPassword)}
			service["volumes"] = []string{"redis_data:/bitnami/redis/data"}
			(*rawCompose)["volumes"].(map[string]interface{})["redis_data"] = map[string]string{
				"driver": "local",
			}
		}else{
			service["environment"] = []string{fmt.Sprintf("CASSANDRA_PASSWORD=%s", resource.DbPassword), fmt.Sprintf("CASSANDRA_PASSWORD_SEEDER=yes"), fmt.Sprintf("CASSANDRA_CQL_PORT_NUMBER=%v", resource.Port)}
			service["ports"] = []string{fmt.Sprintf("%v:%v", resource.Port, resource.Port)}
			service["volumes"] = []string{"cassandra_data:/bitnami"}
			(*rawCompose)["volumes"].(map[string]interface{})["cassandra_data"] = map[string]string{
				"driver": "local",
			}
		}
	}

	(*rawCompose)["services"].(map[string]interface{})[resource.Name] = service
}

var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs test all - test your entire build environment in docker locally before deploying",
	Long: `kubefs test all - test your entire build environment in docker locally before deploying
example:
	kubefs test all --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Testing all resources in docker")
		
		for _, resource := range utils.ManifestData.Resources {
			modifyRawCompose(&rawCompose, &resource,)
		}

		fileErr := utils.WriteYaml(&rawCompose, "docker-compose.yaml")
		if fileErr == types.ERROR {
			utils.PrintError("Error writing docker-compose.yaml file")
			return
		}

		var onlyWrite bool
		var persist bool
		onlyWrite, _ = cmd.Flags().GetBool("only-write")
		persist, _ = cmd.Flags().GetBool("persist-data")

		if !onlyWrite {
			command := exec.Command("sh", "-c", "docker compose up")
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err := command.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error running docker compose: %v", err))
				return
			}

			if persist{
				command = exec.Command("sh", "-c", "docker compose down")
			}else{
				command = exec.Command("sh", "-c", "docker compose down -v --rmi all")
			}

			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err = command.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error stopping docker compose: %v", err))
				return
			}
		}
	},
}

var testResourceCmd = &cobra.Command{
	Use:   "resource [name, ...]",
	Short: "kubefs test resource [name, ...] - test listed resource in docker locally before deploying",
	Long: `kubefs test resource [name ...] - test listed resource in docker locally before deploying
example:
	kubefs test resource my-api my-frontend my-database --flags,
	kubefs test resource my-api --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		var names = args

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		utils.PrintWarning(fmt.Sprintf("Testing resources %v in docker", names))

		for _, name := range names {

			var resource *types.Resource
			resource = utils.GetResourceFromName(name)

			if resource == nil {
				utils.PrintError(fmt.Sprintf("Resource %s not found", name))
				break
			}

			modifyRawCompose(&rawCompose, resource)
		}

		fileErr := utils.WriteYaml(&rawCompose, "docker-compose.yaml")
		if fileErr == types.ERROR {
			utils.PrintError("Error writing docker-compose.yaml file")
			return
		}

		var onlyWrite bool
		var persist bool
		onlyWrite, _ = cmd.Flags().GetBool("only-write")
		persist, _ = cmd.Flags().GetBool("persist-data")

		if !onlyWrite {
			command := exec.Command("sh", "-c", "docker compose up")
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err := command.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error running docker compose: %v", err))
				return
			}

			if persist{
				command = exec.Command("sh", "-c", "docker compose down")
			}else{
				command = exec.Command("sh", "-c", "docker compose down -v --rmi all")
			}

			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err = command.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error stopping docker compose: %v", err))
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testAllCmd)
	testCmd.AddCommand(testResourceCmd)

	testCmd.PersistentFlags().BoolP("only-write", "w", false, "only create the docker compose file; dont start it")
	testCmd.PersistentFlags().BoolP("persist-data", "p", false, "persist images & volume data after testing")
}
