/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"errors"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile [command]",
	Short: "kubefs compile - build and push docker images for resources",
	Long: `kubefs compile - build and push docker images for resources
example: 
	kubefs compile all --flags,
	kubefs compile resource <frontend> <api> <database> --flags,
	kubefs compile resource <frontend> --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func compileUnique(resource *types.Resource, onlyBuild bool, onlyPush bool) error {
	if resource.Type == "database" {
		return errors.New("Database resources cannot be compiled")
	}
	
	var commands []string
	
	if !onlyPush {
		// build docker image
		commands = append(commands, fmt.Sprintf("cd %s && echo 'node_modules\nDockerfile\ndocker-compose.yaml\n.*\ndeploy/' > .dockerignore; echo ''", resource.Name))
		if resource.Type== "api"{
			// api
			if resource.Framework == "nest"{
				// nest
				commands = append(commands, fmt.Sprintf("cd %s && echo 'dist' >> .dockerignore && echo 'FROM node:alpine\n\nWORKDIR /usr/src/app\n\nCOPY package*.json ./\nRUN npm install\n\nCOPY . .\n\nRUN npm run build\n\nEXPOSE %v\nENV PORT=%v\nCMD [\"node\",\"dist/main\"]' > Dockerfile", resource.Name, resource.Port, resource.Port))
			}else if resource.Framework == "fast"{
				// fast
				commands = append(commands,
					fmt.Sprintf("cd %s && source venv/bin/activate && pip freeze > requirements.txt && deactivate", resource.Name),
					fmt.Sprintf("cd %s && echo 'venv' >> .dockerignore && echo 'FROM python:slim\n\nWORKDIR /app\n\nCOPY requirements.txt .\nRUN pip install -r requirements.txt\n\nCOPY . .\n\nEXPOSE %v\nCMD [\"uvicorn\", \"main:app\", \"--host\", \"0.0.0.0\", \"--port\", \"%v\"]' > Dockerfile", resource.Name, resource.Port, resource.Port),
				)
			}else{
				// go
				commands = append(commands, fmt.Sprintf("cd %s && echo 'FROM golang:alpine\n\nWORKDIR /app\n\nCOPY go.mod go.sum ./\n\nRUN go mod download\n\nCOPY . .\n\nRUN go build -o %s .\n\nEXPOSE %v\n\nCMD [\"./%s\"]' > Dockerfile", resource.Name, resource.Name, resource.Port, resource.Name))
			}
		}else{
			// frontend
			if resource.Framework == "next"{
				// next js
				commands = append(commands, fmt.Sprintf("cd %s && echo 'FROM node:alpine\n\nWORKDIR /app\n\nCOPY package.json package-lock.json ./\n\nRUN npm install\n\nCOPY . .\n\nRUN npm run build\n\nEXPOSE %v\n\nENV PORT=%v\n\nCMD [\"npm\", \"run\", \"start\"]' > Dockerfile", resource.Name, resource.Port, resource.Port))
			}else if resource.Framework == "remix"{
				// remix
				commands = append(commands, fmt.Sprintf("cd %s && echo 'build/' >> .dockerignore && echo 'FROM node:alpine\n\nWORKDIR /app\n\nCOPY package.json package-lock.json ./\n\nRUN npm install\n\nCOPY . .\n\nRUN npm run build\n\nEXPOSE %v\n\nENV PORT=%v\n\nCMD [\"npm\", \"run\", \"start\"]' > Dockerfile", resource.Name, resource.Port, resource.Port))
			}else{
				// svelte
				commands = append(commands, fmt.Sprintf("cd %s && echo 'FROM node:alpine\n\nWORKDIR /app\n\nCOPY package.json package-lock.json ./\n\nRUN npm install\n\nCOPY . .\n\nRUN npm run build\n\nEXPOSE %v\n\nCMD [\"npm\",\"run\", \"preview\", \"--\", \"--port\", \"%v\", \"--host\"]' > Dockerfile", resource.Name, resource.Port, resource.Port))
			}
		}

		commands = append(commands, fmt.Sprintf("cd %s && docker build -t %s:latest .", resource.Name, resource.DockerRepo))

		err := utils.RunMultipleCommands(commands, true, true)
		if err != nil {
			return err
		}

	}

	if !onlyBuild {
		// push docker image
		err := utils.RunCommand(fmt.Sprintf("docker images | grep %s", resource.DockerRepo), true, true)
		if err != nil {
			return err
		}

		utils.PrintWarning(fmt.Sprintf("Pushing docker image for resource %s", resource.Name))

		err = utils.RunCommand(fmt.Sprintf("docker push %s:latest", resource.DockerRepo), true, true)
		if err != nil {
			return err
		}
	}

	return nil
}


var compileAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs compile all - build and push docker images for all resources",
	Long: `kubefs compile - build and push docker images for all resources
example: 
	kubefs compile all --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyBuild, onlyPush bool
		onlyBuild, _ = cmd.Flags().GetBool("only-build")
		onlyPush, _ = cmd.Flags().GetBool("only-push")

		var errors []string
		var successes []string

        utils.PrintWarning("Compiling all resources")

		for _, resource := range utils.ManifestData.Resources {
			err := compileUnique(&resource, onlyBuild, onlyPush)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error compiling resource %s. %v", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}
			successes = append(successes, resource.Name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error compiling resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v compiled successfully", successes))
		}

	},
}	

var compileResourceCmd = &cobra.Command{
	Use:   "resource [name ...]",
	Short: "kubefs compile resource [name ...] - build and push docker images for listed resources",
	Long: `kubefs compile resource [name ...] - build and push docker images for listed resources
example: 
	kubefs compile resource <frontend> <api> <database> --flags,
	kubefs compile resource <frontend> --flags,
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

		var onlyBuild, onlyPush bool
		onlyBuild, _ = cmd.Flags().GetBool("only-build")
		onlyPush, _ = cmd.Flags().GetBool("only-push")

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Compiling resource %v", args))

		for _, name := range args{
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = compileUnique(resource, onlyBuild, onlyPush)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error compiling resource %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
			}

			successes = append(successes, name)

		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error compiling resources %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v compiled successfully", successes))
		}
	},
}	


func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.AddCommand(compileAllCmd)
	compileCmd.AddCommand(compileResourceCmd)

	compileCmd.PersistentFlags().BoolP("only-build", "b", false, "only build the docker image for resource")
	compileCmd.PersistentFlags().BoolP("only-push", "p", false, "only push the docker image for resource")
}
