/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"os/exec"
	"os"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile [command]",
	Short: "kubefs compile - build and push docker images for resources",
	Long: `kubefs compile - build and push docker images for resources`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func compileUnique(resource *types.Resource, onlyBuild bool, onlyPush bool) (int){
	
	var commands []string
	
	if !onlyPush {
		// build docker image
		utils.PrintWarning(fmt.Sprintf("Building docker image for resource %s", resource.Name))

		cmd := exec.Command("sh", "-c", fmt.Sprintf("(cd %s; rm Dockerfile; rm .dockerignore; docker rmi %s:latest; touch .dockerignore; echo 'deploy/\nkubefs.env' >> .dockerignore; echo '')", resource.Name, resource.DockerRepo))
		err := cmd.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error removing docker image: %v", err))
			return types.ERROR
		}

		if resource.Type == "api" {
			// api
			if resource.Framework == "koa" {
				// koa
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM node:alpine\nWORKDIR /app\nCOPY package*.json ./\nRUN yarn install --production\nCOPY . .\nEXPOSE %v\nENV NODE_ENV=production\nCMD [\"node\", \"index.js\"]' > Dockerfile)", resource.Name, resource.Port),
				)
			} else if resource.Framework == "fast" {
				// fast
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM python:slim\nWORKDIR /app\nCOPY requirements.txt .\nRUN pip install -r requirements.txt\nCOPY . .\nEXPOSE %v\nCMD [\"uvicorn\", \"main:app\", \"--host\", \"0.0.0.0\", \"--port\", \"%v\"]' > Dockerfile)", resource.Name, resource.Port, resource.Port),
				)
			}else{
				// go
				commands = append(commands,
					fmt.Sprintf("(cd %s && source venv/bin/activate && pip freeze > requirements.txt && deactivate)", resource.Name),
					fmt.Sprintf("(cd %s && echo 'FROM golang:alpine\n\nWORKDIR /app\n\nCOPY go.mod go.sum ./\n\nRUN go mod download\n\nCOPY . .\n\nRUN go build -o %s .\n\nEXPOSE %v\n\n# Command to run the executable\nCMD [\"./%s\"]' > Dockerfile)", resource.Name, resource.Name, resource.Port, resource.Name),
				)
			}

			commands = append(commands,
				fmt.Sprintf("(cd %s && echo 'Dockerfile\ndocker-compose.yaml\n' >> .dockerignore )", resource.Name),
				fmt.Sprintf("(cd %s && docker buildx build -t %s:latest .)", resource.Name, resource.DockerRepo),
			)

		} else if resource.Type == "frontend" {
			// frontend
			if resource.Framework == "react" {
				// react
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM node:alpine AS builder\nWORKDIR /app\nCOPY package.json yarn.lock ./\nRUN yarn install --frozen-lockfile --silent\nCOPY . .\nRUN yarn build\n\nFROM nginx:alpine\nCOPY --from=builder /app/build /usr/share/nginx/html\nEXPOSE 80\nCMD [\"nginx\", \"-g\", \"daemon off;\"]' > Dockerfile)", resource.Name),
				)
			} else if resource.Framework == "angular" {
				// angular
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM node:alpine AS builder\nWORKDIR /app\nCOPY package*.json ./\nRUN npm install --silent\nCOPY . .\nRUN npm run build\n\nFROM nginx:alpine\nCOPY --from=builder /app/dist/%s/browser /usr/share/nginx/html\nEXPOSE 80\nCMD [\"nginx\", \"-g\", \"daemon off;\"]' > Dockerfile)", resource.Name, resource.Name),
				)
			}else{
				// vue
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM node:alpine AS builder\nWORKDIR /app\nCOPY package*.json ./\nRUN npm install --silent\nCOPY . .\nRUN npm run build\n\nFROM nginx:alpine\nCOPY --from=builder /app/dist /usr/share/nginx/html\nEXPOSE 80\nCMD [\"nginx\", \"-g\", \"daemon off;\"]' > Dockerfile)", resource.Name),
				)
			}

			commands = append(commands,
				fmt.Sprintf("(cd %s && echo 'node_modules\n.gitignore\nDockerfile\ndocker-compose.yaml\nREADME.md' >> .dockerignore )", resource.Name),
				fmt.Sprintf("(cd %s && docker buildx build -t %s:latest .)", resource.Name, resource.DockerRepo),
			)
		} else {
			// database nothing to do 
			return types.SUCCESS
		}

		for _, command := range commands {
			// run command
			cmd = exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error building docker image: %v", err))
				return types.ERROR
			}
		}

	}

	if !onlyBuild {
		// push docker image
		if resource.DockerRepo == ""{
			utils.PrintError(fmt.Sprintf("Docker repo not found for resource %s", resource.Name))
			return types.ERROR
		}

		utils.PrintWarning(fmt.Sprintf("Pushing docker image for resource %s to %s:latest", resource.Name, resource.DockerRepo))

		cmd := exec.Command("sh", "-c", fmt.Sprintf("(docker push %s:latest)",resource.DockerRepo))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error pushing docker image: %v", err))
			return types.ERROR
		}
	}

	return types.SUCCESS
}


var compileAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs compile all - build and push docker images for all resources",
	Long: `kubefs compile - build and push docker images for all resources`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var onlyBuild, onlyPush bool
		onlyBuild, _ = cmd.Flags().GetBool("only-build")
		onlyPush, _ = cmd.Flags().GetBool("only-push")

        utils.PrintWarning("Compiling all resources")

		for _, resource := range utils.ManifestData.Resources {
			err := compileUnique(&resource, onlyBuild, onlyPush)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error compiling resource %s", resource.Name))
				return 
			}
		}

		utils.PrintSuccess("All resources compiled successfully")
	},
}	

var compileResourceCmd = &cobra.Command{
	Use:   "resource [name]",
	Short: "kubefs compile resource [name] - build and push docker images for a unique resources",
	Long: `kubefs compile resource [name] - build and push docker images for a unique resources`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var onlyBuild, onlyPush bool
		onlyBuild, _ = cmd.Flags().GetBool("only-build")
		onlyPush, _ = cmd.Flags().GetBool("only-push")

		name := args[0]
		utils.PrintWarning(fmt.Sprintf("Compiling resource %s", name))

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

		err := compileUnique(resource, onlyBuild, onlyPush)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error compiling resource %s", name))
			return
		}

		utils.PrintSuccess(fmt.Sprintf("Resource %s compiled successfully", name))
	},
}	


func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.AddCommand(compileAllCmd)
	compileCmd.AddCommand(compileResourceCmd)

	compileCmd.PersistentFlags().BoolP("only-build", "b", false, "only build the docker image for resource")
	compileCmd.PersistentFlags().BoolP("only-push", "p", false, "only push the docker image for resource")
}
