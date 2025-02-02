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
	Long: `kubefs test - test your build environment in docker locally before deploying`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var rawCompose = map[string]interface{}{
	"services": map[string]interface{}{
		"kubefsHelper": map[string]interface{}{
		"image": "rmedicharla/kubefshelper:latest",
		"labels": []string{
			"traefik.enable=true",
			"traefik.http.routers.backend.rule=PathPrefix(`/env`) || PathPrefix(`/api`)",
			"traefik.http.services.backend.loadbalancer.server.port=6000",
		},
		"networks": []string{
			"shared_network",
		},
		"environment": []string{},
		},
		"traefik": map[string]interface{}{
		"image": "traefik:latest",
		"command": []string{
			"--api.insecure=true",
			"--providers.docker=true",
			"--entrypoints.web.address=:80",
			"--api.dashboard=true",
		},
		"networks": []string{
			"shared_network",
		},
		"ports": []string{
			"80:80",
			"8080:8080",
		},
		"volumes": []string{
			"/var/run/docker.sock:/var/run/docker.sock:ro",
		},
		},
	},
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

	if resource.Type == "api" {
		service["labels"] = []string{
			"traefik.enable=false",
		}
		service["ports"] = []string{
			fmt.Sprintf("%v:%v", resource.Port, resource.Port),
		}

		for _, r := range utils.ManifestData.Resources {
			service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%sHOST=%s", r.Name, r.DockerHost))
		}

	}else if resource.Type == "frontend" {
		service["labels"] = []string{
			"traefik.enable=true",
			fmt.Sprintf("traefik.http.routers.%s.rule=PathPrefix(`/`)", resource.Name),
			fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=80", resource.Name),
		}
	}else{
		service["labels"] = []string{
			"traefik.enable=false",
		}

		if resource.Framework == "redis" {
			service["environment"] = []string{fmt.Sprintf("REDIS_PASSWORD=%s", resource.DbPassword)}
			service["ports"] = []string{fmt.Sprintf("%v:%v", resource.Port, resource.Port)}
			service["command"] = []string{"redis-server", "--port", fmt.Sprintf("%v", resource.Port), "--requirepass", resource.DbPassword}
			service["volumes"] = []string{"redis_data:/bitnami/redis/data"}
			(*rawCompose)["volumes"].(map[string]interface{})["redis_data"] = map[string]string{
				"driver": "local",
			}
		}else{
			service["environment"] = []string{fmt.Sprintf("CASANDRA_USER=%s", resource.DbUsername), fmt.Sprintf("CASSANDRA_PASSWORD=%s", resource.DbPassword)}
			service["ports"] = []string{fmt.Sprintf("%v:9042", resource.Port)}
			service["volumes"] = []string{"cassandra_data:/bitnami"}
			(*rawCompose)["volumes"].(map[string]interface{})["cassandra_data"] = map[string]string{}
		}
	}

	(*rawCompose)["services"].(map[string]interface{})[resource.Name] = service
	(*rawCompose)["services"].(map[string]interface{})["kubefsHelper"].(map[string]interface{})["environment"] = append((*rawCompose)["services"].(map[string]interface{})["kubefsHelper"].(map[string]interface{})["environment"].([]string), fmt.Sprintf("%sHOST=%s", resource.Name, resource.DockerHost))
}

var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs test all - test your entire build environment in docker locally before deploying",
	Long: `kubefs test all - test your entire build environment in docker locally before deploying`,
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

		utils.PrintWarning("View your frontend resources at http://localhost\n View the traefik dashboard at http://localhost:8080")

		command := exec.Command("sh", "-c", "docker compose up")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err := command.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error running docker compose: %v", err))
			return
		}

		command = exec.Command("sh", "-c", "docker compose down -v --rmi all")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error stopping docker compose: %v", err))
			return
		}
	},
}

var testResourceCmd = &cobra.Command{
	Use:   "resource [name]",
	Short: "kubefs test resource - test a specific resource in docker locally before deploying",
	Long: `kubefs test resource - test a specific resource in docker locally before deploying`,
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
        utils.PrintWarning(fmt.Sprintf("Removing resource %s", name))

		var resource *types.Resource
		resource = utils.GetResourceFromName(name)

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
			return
		}

		modifyRawCompose(&rawCompose, resource)

		fileErr := utils.WriteYaml(&rawCompose, fmt.Sprintf("%s/docker-compose.yaml", resource.Name))
		if fileErr == types.ERROR {
			utils.PrintError("Error writing docker-compose.yaml file")
			return
		}

		utils.PrintWarning("View your frontend resources at http://localhost\n View the traefik dashboard at http://localhost:8080")

		command := exec.Command("sh", "-c", fmt.Sprintf("(cd %s && docker compose up)", resource.Name))
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err := command.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error running docker compose: %v", err))
			return
		}

		command = exec.Command("sh", "-c", fmt.Sprintf("(cd %s && docker compose down -v --rmi all)", resource.Name))
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error stopping docker compose: %v", err))
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testAllCmd)
	testCmd.AddCommand(testResourceCmd)
}
