/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "kubefs run - run a resource locally or in the docker containers",
	Long: `kubefs run - run a resource locally or in the docker containers`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func runKubefsHelper(ctx context.Context, project *types.Project, wg *sync.WaitGroup){
	defer wg.Done()

	var commands []string

	commands = append(commands, "rm .kubefshelper/.env; touch .kubefshelper/.env")
	for _, resource := range project.Resources {
		commands = append(commands, fmt.Sprintf("echo %sHOST=%s >> .kubefshelper/.env", resource.Name, resource.LocalHost))
	}
	commands = append(commands, "(cd .kubefshelper && ./kubefsHelper)")

	for _, command := range commands{
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil && ctx.Err() == nil {
			utils.PrintError(fmt.Sprintf("Error running kubefs-helper: %v", err))
		}
	}
}

func runUnique(ctx context.Context, project *types.Project, resource *types.Resource, platform string, wg *sync.WaitGroup){
	defer wg.Done()

	if platform == "local" {
		cmd := exec.CommandContext(ctx, "sh", "-c", resource.UpLocal)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil && ctx.Err() == nil {
			utils.PrintError(fmt.Sprintf("Error running resource locally: %v", err))
		}
	}else{
		// Handle docker platform if needed
		_, err := os.Stat(fmt.Sprintf("%s/docker-compose.yaml", resource.Name))
		if os.IsNotExist(err) {
			utils.PrintError(fmt.Sprintf("docker-compose.yaml file not found for resource: %s", resource.Name))
			return
		}

		if resource.Type == "frontend" || resource.Type == "api" {
			fileErr, composeFile := utils.ReadYaml(fmt.Sprintf("%s/docker-compose.yaml", resource.Name))
			if fileErr == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error reading docker-compose.yaml file for resource: %s", resource.Name))
				return 
			}
			for _, r := range project.Resources {
				composeFile["services"].(map[string]interface{})["backend"].(map[string]interface{})["environment"] = append(composeFile["services"].(map[string]interface{})["backend"].(map[string]interface{})["environment"].([]interface{}), fmt.Sprintf("%sHOST=%s", r.Name, r.DockerHost))
			}
			fileErr = utils.WriteYaml(&composeFile, fmt.Sprintf("%s/docker-compose.yaml", resource.Name))
			if fileErr == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error updating docker-compose.yaml file for resource: %s", resource.Name))
				return
			}
		}

		cmd := exec.CommandContext(ctx, "sh", "-c", "docker network inspect shared_network")
		cmd.Run()
		if cmd.ProcessState.ExitCode() != 0 {
			utils.PrintWarning("Creating shared network for docker resources")
			cmd = exec.CommandContext(ctx, "sh", "-c", "docker network create shared_network")
			cmd.Run()
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", resource.UpDocker)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil && ctx.Err() == nil {
			utils.PrintError(fmt.Sprintf("Error running resource in Docker: %v", err))
		}
	}
}

var runAllCmd = &cobra.Command{
    Use:   "all",
    Short: "kubefs run all - run all resources locally or in the docker containers",
    Long:  "kubefs run all - run all resources locally or in the docker containers",
    Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

        utils.PrintWarning("Running all resources")

		var platform string
		platform, _ = cmd.Flags().GetString("platform")

		if platform != "docker" && platform != "local" {
			utils.PrintError("Invalid platform: use 'local' or 'docker'")
			return
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle interrupt signal (Ctrl+C)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\nReceived interrupt signal, shutting down...")
			cancel()
		}()

		if platform == "local"{
			wg.Add(1)
			go runKubefsHelper(ctx, &utils.ManifestData, &wg)
		}

        for _, resource := range utils.ManifestData.Resources {
			if platform == "local" && resource.Type == "database" {
				utils.PrintError("Docker platform not supported for database resources")
				break
			}
			wg.Add(1)
			go runUnique(ctx, &utils.ManifestData, &resource, platform, &wg)
        }

		wg.Wait()
    },
}

var runResourceCmd = &cobra.Command{
    Use:   "resource [name]",
    Short: "kubefs run resource [name] - run a specific resource locally or in the docker containers",
    Long:  "kubefs run resource [name] - run a specific resource locally or in the docker containers",
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

		var platform string
		platform, _ = cmd.Flags().GetString("platform")

		if platform != "docker" && platform != "local" {
			utils.PrintError("Invalid platform: use 'local' or 'docker'")
			return
		}

		var resource *types.Resource
		resource = utils.GetResourceFromName(name)

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
			return
		}

		if platform == "local" && resource.Type == "database" {
			utils.PrintError("Docker platform not supported for database resources")
			return
		}

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle interrupt signal (Ctrl+C)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\nReceived interrupt signal, shutting down...")
			cancel()
		}()

		if platform == "local"{
			wg.Add(1)
			go runKubefsHelper(ctx, &utils.ManifestData, &wg)
		}

		wg.Add(1)
		go runUnique(ctx, &utils.ManifestData, resource, platform, &wg)

		wg.Wait()
    },
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runAllCmd)
	runCmd.AddCommand(runResourceCmd)

	runCmd.PersistentFlags().StringP("platform", "p", "local", "Choose the platform to run the resource on [local, docker]")
}
