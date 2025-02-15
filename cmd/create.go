/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/zalando/go-keyring"
	"strings"
	"errors"
)

// createCmd represents the create command

var resourcePort int
var resourceFramework string
var resourceName string

var createCmd = &cobra.Command{
	Use:   "create [command]",
	Short: "kubefs create - easily create backend, frontend, & db resources to be used within your application",
	Long: `kubefs create - easily create backend, frontend, & db resources to be used within your application
example:
	kubefs create api <api name> --flags
	kubefs create frontend <frontend name> --flags
	kubefs create database <database name> --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func parseInfo(cmd *cobra.Command,args []string, resource string) error {
	if len(args) < 1 {
		cmd.Help()
		return errors.New("Please provide a name for the resource")
	}

	name := args[0]
	if err := utils.VerifyName(name); err != nil {
		return err
	}
	resourceName = name
	
	port, _ := cmd.Flags().GetInt("port")
	if err := utils.VerifyPort(port); err != nil {
		return err
	}
	resourcePort = port

	framework, _ := cmd.Flags().GetString("framework")
	if err := utils.VerifyFramework(framework, resource); err != nil {
		return err
	}
	resourceFramework = framework

	utils.PrintWarning(fmt.Sprintf("Creating %s named %s on port %v using the %s framework\n", resource, name, port, framework))
	return nil
}

func createDockerRepo(name string) (string, error) {
	utils.PrintWarning(fmt.Sprintf("Creating Docker Repository for %s", name))
	desc, err := utils.ReadInput("Enter resource description: ", true)
	if err != nil {
		return "", err
	}

	creds, err := keyring.Get("docker", "kubefs")
	if err != nil {
		return "", err
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
		return "", err
	}	

	_, err = utils.PostRequest(types.DOCKER_REPO_ENDPOINT, 
		map[string]string{
			"Content-Type": "application/json",
			"Authorization": fmt.Sprintf("JWT %s", response.Token),
		}, map[string]interface{}{
			"name": name,
			"namespace": username,
			"is_private": false,
			"full_description": desc,
			"description": desc,
		},
	)
	if err != nil{
		return "", err
	}

	return fmt.Sprintf("%s/%s", username, name), nil
}

var createApiCmd = &cobra.Command{
	Use:   "api [name]",
	Short: "kubefs create api - create a new API resource",
	Long: `kubefs create api - create a new API resource
example: 
	kubefs create api <name> --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil{
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}
		
		if err := parseInfo(cmd, args, "api"); err != nil {
			utils.PrintError(err.Error())
			return
		}

		var commands []string
		var upLocal string

		if resourceFramework == "fast" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && python3 -m venv venv && source venv/bin/activate && pip install \"fastapi[standard]\" python-dotenv && pip freeze > requirements.txt && deactivate && touch main.py", resourceName),
				fmt.Sprintf("cd %s && echo 'from fastapi import FastAPI\napp = FastAPI()\n#KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n@app.get(\"/health\")\nasync def root():\n\treturn {\"status\": \"ok\"}' > main.py", resourceName),
			}

			upLocal = fmt.Sprintf("source venv/bin/activate && uvicorn main:app --reload --port %v", resourcePort)
		}else if resourceFramework == "nest" {
			commands = []string{
				fmt.Sprintf("npx -p @nestjs/cli nest new %s -g -p npm", resourceName),
				fmt.Sprintf("cd %s/src/ && head -n 11 app.controller.ts > temp && mv temp app.controller.ts && echo '\t//KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n\t@Get(\"/health\")\n\tgetHealth(): string {\n\t\t return \"ok\";\n\t}\n}' >> app.controller.ts", resourceName),
			}

			upLocal = fmt.Sprintf("PORT=%v npm run start:debug", resourcePort)
		}else{
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && go mod init %s && go get -u github.com/gorilla/mux", resourceName, resourceName),
				fmt.Sprintf("cd %s && echo 'package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n\t\"github.com/gorilla/mux\"\n)\n\nfunc main() {\n\tr := mux.NewRouter()\n\t//KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n\tr.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"ok\")\n\t})\n\tfmt.Println(\"Listening on Port %v\")\n\thttp.ListenAndServe(\":%v\", r)\n}' > main.go", resourceName, resourcePort, resourcePort),
			}

			upLocal = fmt.Sprintf("go run main.go")
		}

		err := utils.RunMultipleCommands(commands, true, true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating resource. %v", err.Error()))
			return
		}

		dockerRepo, err := createDockerRepo(resourceName)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating docker repo. %v", err.Error()))
			return
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "api", 
			Framework:resourceFramework, 
			UpLocal: upLocal, 
			LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), 
			DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), 
			DockerRepo: dockerRepo, 
			ClusterHost: fmt.Sprintf("http://%s-deploy.%s.svc.cluster.local", resourceName, resourceName),
		})
		
		if err := utils.WriteManifest(&utils.ManifestData, "manifest.yaml"); err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error writing manifest. %v", err.Error()))
			return
		}

		utils.PrintSuccess(fmt.Sprintf("Successfully created API %s on port %v using the %s framework", resourceName, resourcePort, resourceFramework))
	},
}

var createFrontendCmd = &cobra.Command{
	Use:   "frontend [name]",
	Short: "kubefs create frontend - create a new frontend resource",
	Long: `kubefs create frontend - create a new frontend resource
example:
	kubefs create frontend <name> --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil{
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		if err := parseInfo(cmd, args, "frontend"); err != nil {
			utils.PrintError(err.Error())
			return
		}

		var commands []string
		var startCommand string

		if resourceFramework == "next" {
			commands = []string{
				fmt.Sprintf("npx create-next-app@latest %s --ts --yes ", resourceName),
				fmt.Sprintf("cd %s && rm -rf .git", resourceName),
			}

			startCommand = fmt.Sprintf("next dev --turbopack --port %v", resourcePort)
		}else if resourceFramework == "remix" {
			commands = []string{
				fmt.Sprintf("npx create-remix@latest %s --no-git-init --yes", resourceName),
			}

			startCommand = fmt.Sprintf("remix vite:dev --port %v", resourcePort)
		}else{
			commands = []string{
				fmt.Sprintf("npx sv create --template minimal --types ts --no-add-ons --no-install %s", resourceName),
				fmt.Sprintf("cd %s && npm i", resourceName),
			}
			startCommand = fmt.Sprintf("vite dev --port %v", resourcePort)
		}

		err := utils.RunMultipleCommands(commands, true, true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating resource. %v", err.Error()))
			return
		}

		packageJson, err := utils.ReadJson(fmt.Sprintf("%s/package.json", resourceName))
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading package.json. %v", err.Error()))
			return
		}

		(*packageJson)["scripts"].(map[string]interface{})["dev"] = startCommand

		err = utils.WriteJson((*packageJson), fmt.Sprintf("%s/package.json", resourceName))
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error writing package.json. %v", err.Error()))
			return
		}

		dockerRepo, err := createDockerRepo(resourceName)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating docker repo. %v", err.Error()))
			return
		}

		hostDomain, err := utils.ReadInput("Enter the host domain the ingresss should accept: (*) for all : ", true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
			return
		}

		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "frontend", 
			Framework:resourceFramework, 
			UpLocal: "npm run dev", 
			LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), 
			DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), 
			DockerRepo: dockerRepo, 
			ClusterHost: fmt.Sprintf("http://%s-deploy.%s.svc.cluster.local", resourceName, resourceName),
			Opts: map[string]string{
				"host-domain": hostDomain,
			},
		})
		
		err = utils.WriteManifest(&utils.ManifestData, "manifest.yaml")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error writing manifest. %v", err.Error()))
			return
		}

		utils.PrintSuccess(fmt.Sprintf("Successfully created frontend %s on port %v using the %s framework", resourceName, resourcePort, resourceFramework))
	},
}

var createDbCmd = &cobra.Command{
	Use:   "database [name]",
	Short: "kubefs create database - create a new database resource",
	Long: `kubefs create database - create a new database resource
example:
	kubefs create database <db> --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil{
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}
		
		if err := parseInfo(cmd, args, "database"); err != nil {
			utils.PrintError(err.Error())
			return
		}

		dockerRepo := fmt.Sprintf("bitnami/%s", resourceFramework)
		
		var clusterHost string
		var clusterHostRead string
		if resourceFramework == "postgresql" {
			clusterHost = fmt.Sprintf("http://%s-postgresql-primary.%s.svc.cluster.local", resourceName, resourceName)
			clusterHostRead = fmt.Sprintf("http://%s-postgresql-read.%s.svc.cluster.local", resourceName, resourceName)		
		}else{
			clusterHost = fmt.Sprintf("http://%s-redis-master.%s.svc.cluster.local", resourceName, resourceName)
			clusterHostRead = fmt.Sprintf("http://%s-redis-replicas.%s.svc.cluster.local", resourceName, resourceName)
		}

		err := utils.RunCommand(fmt.Sprintf("mkdir %s", resourceName), true, true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating resource. %v", err.Error()))
			return
		}

		password, err := utils.ReadInput("Enter a password for the database: ", true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
			return
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "database", 
			Framework:resourceFramework, 
			LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), 
			DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), 
			DockerRepo: dockerRepo, 
			ClusterHost: clusterHost, 
			ClusterHostRead: clusterHostRead,
			Opts: map[string]string{
				"password": password,
				"default-database": "default",
			},
		})

		err = utils.WriteManifest(&utils.ManifestData, "manifest.yaml")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error writing manifest. %v", err.Error()))
			return
		}

		utils.PrintWarning(fmt.Sprintf("Creating database with '%s' as password. Store this to interact with the database", password))
		utils.PrintSuccess(fmt.Sprintf("Successfully created database %s on port %v using the %s framework", resourceName, resourcePort, resourceFramework))
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createApiCmd)
	createCmd.AddCommand(createFrontendCmd)
	createCmd.AddCommand(createDbCmd)
	createApiCmd.Flags().StringP("framework", "f", "fast", "Framework to use for API [fast | nest | go]")
	createFrontendCmd.Flags().StringP("framework", "f", "next", "Framework to use for Frontend [next | remix | sveltekit]")
	createDbCmd.Flags().StringP("framework", "f", "postgresql", "Type of database to use [postgresql | redis]")

	createCmd.PersistentFlags().IntP("port", "p", 3000, "Specific port to be used")
}
