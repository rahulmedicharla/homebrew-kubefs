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

func runUnique(ctx context.Context, resource *types.Resource, platform string, wg *sync.WaitGroup){
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
		err := cmd.Run()
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
		
        for _, resource := range utils.ManifestData.Resources {
			wg.Add(1)
			go runUnique(ctx, &resource, platform, &wg)
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
		for _, r := range utils.ManifestData.Resources {
			if r.Name == name {
				resource = &r
				break
			}
		}

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
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

		wg.Add(1)
		go runUnique(ctx, resource, platform, &wg)

		wg.Wait()
    },
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runAllCmd)
	runCmd.AddCommand(runResourceCmd)

	runCmd.PersistentFlags().StringP("platform", "p", "local", "Choose the platform to run the resource on [local, docker]")
}
