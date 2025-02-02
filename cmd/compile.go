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

func compileUnique(resource *types.Resource, onlyBuild bool, onlyPush bool) (int, string){
	
	var commands []string
	var up_docker string
	
	if !onlyPush {
		// build docker image
		utils.PrintWarning(fmt.Sprintf("Building docker image for resource %s", resource.Name))

		cmd := exec.Command("sh", "-c", fmt.Sprintf("(cd %s; rm Dockerfile; rm .dockerignore; docker rmi %s:latest; rm docker-compose.yaml; touch .dockerignore; echo 'deploy/' >> .dockerignore; echo '')", resource.Name, resource.DockerRepo))
		err := cmd.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error removing docker image: %v", err))
			return types.ERROR, ""
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
					fmt.Sprintf("(cd %s && echo 'FROM golang:alpine\n\nWORKDIR /app\n\nCOPY go.mod go.sum ./\n\nRUN go mod download\n\nCOPY . .\n\nRUN go build -o %s .\n\nEXPOSE %v\n\n# Command to run the executable\nCMD [\"./%s\"]' > Dockerfile)", resource.Name, resource.Name, resource.Port, resource.Name),
				)
			}

			commands = append(commands,
				fmt.Sprintf("(cd %s && docker buildx build -t %s:latest .)", resource.Name, resource.DockerRepo),
				fmt.Sprintf("(cd %s && echo 'services:\n  traefik:\n    image: traefik:latest\n    command:\n      - \"--api.insecure=true\"\n      - \"--providers.docker=true\"\n      - \"--entrypoints.web.address=:80\"\n    ports:\n      - \"%v:80\"\n    volumes:\n      - \"/var/run/docker.sock:/var/run/docker.sock:ro\"\n    networks:\n      - shared_network\n\n  api:\n    image: %s:latest\n    labels:\n      - \"traefik.enable=true\"\n      - \"traefik.http.routers.frontend.rule=PathPrefix(`/`)\"\n      - \"traefik.http.services.frontend.loadbalancer.server.port=%v\"\n    networks:\n      - shared_network\n\n  backend:\n    image: rmedicharla/kubefshelper:latest\n    labels:\n      - \"traefik.enable=true\"\n      - \"traefik.http.routers.backend.rule=PathPrefix(`/env`)\"\n      - \"traefik.http.services.backend.loadbalancer.server.port=6000\"\n    networks:\n      - shared_network\n    environment: []\n\nnetworks:\n  shared_network:\n    external: true' > docker-compose.yaml)", resource.Name, resource.Port, resource.DockerRepo, resource.Port), 
				fmt.Sprintf("(cd %s && echo 'Dockerfile\ndocker-compose.yaml\n' >> .dockerignore )", resource.Name),
			)

			up_docker = fmt.Sprintf("(cd %s && docker compose up)", resource.Name)
		} else if resource.Type == "frontend" {
			// frontend
			if resource.Framework == "react" {
				// react
				commands = append(commands,
					fmt.Sprintf("(cd %s && echo 'FROM node:alpine AS builder\nWORKDIR /app\nCOPY package.json yarn.lock ./\nRUN yarn install --frozen-lockfile --silent\nCOPY . .\nRUN npm run build\n\nFROM nginx:alpine\nCOPY --from=builder /app/build /usr/share/nginx/html\nEXPOSE 80\nCMD [\"nginx\", \"-g\", \"daemon off;\"]' > Dockerfile)", resource.Name),
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
				fmt.Sprintf("(cd %s && docker buildx build -t %s:latest .)", resource.Name, resource.DockerRepo),
				fmt.Sprintf("(cd %s && echo 'services:\n  traefik:\n    image: traefik:latest\n    command:\n      - \"--api.insecure=true\"\n      - \"--providers.docker=true\"\n      - \"--entrypoints.web.address=:80\"\n    ports:\n      - \"%v:80\"\n    volumes:\n      - \"/var/run/docker.sock:/var/run/docker.sock:ro\"\n    networks:\n      - shared_network\n\n  frontend:\n    image: %s:latest\n    labels:\n      - \"traefik.enable=true\"\n      - \"traefik.http.routers.frontend.rule=PathPrefix(`/`)\"\n      - \"traefik.http.services.frontend.loadbalancer.server.port=80\"\n    networks:\n      - shared_network\n  backend:\n    image: rmedicharla/kubefshelper:latest\n    labels:\n      - \"traefik.enable=true\"\n      - \"traefik.http.routers.backend.rule=PathPrefix(`/env`) || PathPrefix(`/api`)\"\n      - \"traefik.http.services.backend.loadbalancer.server.port=6000\"\n    networks:\n      - shared_network\n    environment: []\n\nnetworks:\n  shared_network:\n    external: true' > docker-compose.yaml)", resource.Name, resource.Port, resource.DockerRepo),
				fmt.Sprintf("(cd %s && echo 'node_modules\n.gitignore\nDockerfile\ndocker-compose.yaml\nREADME.md' >> .dockerignore )", resource.Name),
			)
			up_docker = fmt.Sprintf("(cd %s && docker compose up)", resource.Name)
		} else {
			// database

			file, err := os.Create(fmt.Sprintf("%s/docker-compose.yaml", resource.Name))
			if err != nil {
				fmt.Println("Error creating file:", err)
				return types.ERROR, ""
			}
			defer file.Close()
		
			if resource.Framework == "cassandra" {
				// cassandra
				_, err = file.WriteString(types.GetCassandraCompose(resource.Port, resource.DbUsername, resource.DbPassword))
				if err != nil {
					fmt.Println("Error writing to file:", err)
					return types.ERROR, ""
				}
			} else {
				// redis
				_, err = file.WriteString(types.GetRedisCompose(resource.Port, resource.DbPassword))
				if err != nil {
					fmt.Println("Error writing to file:", err)
					return types.ERROR, ""
				}
			}

			up_docker = fmt.Sprintf("(cd %s && docker compose up)", resource.Name)

			return types.SUCCESS, up_docker
		}

		for _, command := range commands {
			// run command
			cmd = exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error building docker image: %v", err))
				return types.ERROR, ""
			}
		}

	}

	if !onlyBuild {
		// push docker image
		if resource.DockerRepo == ""{
			utils.PrintError(fmt.Sprintf("Docker repo not found for resource %s", resource.Name))
			return types.ERROR, ""
		}

		utils.PrintWarning(fmt.Sprintf("Pushing docker image for resource %s to %s:latest", resource.Name, resource.DockerRepo))

		cmd := exec.Command("sh", "-c", fmt.Sprintf("(docker push %s:latest)",resource.DockerRepo))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error pushing docker image: %v", err))
			return types.ERROR, ""
		}
	}

	return types.SUCCESS, up_docker
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
			err, up_docker := compileUnique(&resource, onlyBuild, onlyPush)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error compiling resource %s", resource.Name))
				return
			}

			err = utils.UpdateResource(&utils.ManifestData, &resource, "UpDocker" ,up_docker)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error updating resource %s", resource.Name))
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

		err, up_docker := compileUnique(resource, onlyBuild, onlyPush)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error compiling resource %s", name))
			return
		}

		err = utils.UpdateResource(&utils.ManifestData, resource, "UpDocker" ,up_docker)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error updating resource %s", name))
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
