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
	"bufio"
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
	kubefs create api my-api --flags
	kubefs create frontend my-frontend
	kubefs create database my-db
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func parseInfo(cmd *cobra.Command,args []string, resource string) int {
	if len(args) < 1 {
		cmd.Help()
		return types.ERROR
	}

	name := args[0]
	resourceName = name
	port, err := cmd.Flags().GetInt("port")
	if err != nil || port == 6000 || port == 8000 {
		utils.PrintError(fmt.Sprintf(" Invalid port. Port 6000 & 8000 reserved for kubefs: %v", err))
		return types.ERROR
	}
	resourcePort = port

	framework, err := cmd.Flags().GetString("framework")
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error reading framework: %v", err))
		return types.ERROR
	}
	resourceFramework = framework

	allowableFrameworks := types.FRAMEWORKS[resource]
	if (!utils.Contains(allowableFrameworks, framework)) {
		utils.PrintError(fmt.Sprintf("Invalid framework: %s. Allowed frameworks are: %v", framework, allowableFrameworks))
		return types.ERROR
	}

	for _, resource := range utils.ManifestData.Resources {
		if resource.Name == name {
			utils.PrintError(fmt.Sprintf("Resource with name %s already exists", name))
			return types.ERROR
		}
		
		if resource.Port == port {
			utils.PrintError(fmt.Sprintf("Resource with port %v already exists", port))
			return types.ERROR
		}
	}

	utils.PrintWarning(fmt.Sprintf("Creating %s named %s on port %v using the %s framework\n", resource, name, port, framework))
	return types.SUCCESS
}

func createDockerRepo(name string) (int, string) {
	utils.PrintWarning(fmt.Sprintf("Creating Docker Repository for %s", name))
	var input string
	fmt.Print("Enter resource description: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ = reader.ReadString('\n')
	desc := strings.TrimSpace(input)

	url := "https://hub.docker.com/v2/users/login/"

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	creds, err := keyring.Get("docker", "kubefs")
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error getting Docker credentials: %v", err))
		return types.ERROR, ""
	}

	username, pat := strings.Split(creds, ":")[0], strings.Split(creds, ":")[1]

	payload := map[string]interface{}{
		"username": username,
		"password": pat,
	}

	status, response, err := utils.PostRequest(url, headers, payload)
	if status == types.ERROR {
		utils.PrintError(fmt.Sprintf("Error logging into Docker: %v", err))
		return types.ERROR, ""
	}

	if response.Token == "" {
		utils.PrintError(fmt.Sprintf("Error logging into Docker: No token received. %s", response.Detail))
		return types.ERROR, ""
	}

	url = "https://hub.docker.com/v2/repositories/"
	payload = map[string]interface{}{
		"name": name,
		"namespace": username,
		"is_private": false,
		"full_description": desc,
		"description": desc,
	}

	headers = map[string]string{
		"Content-Type": "application/json",
		"Authorization": fmt.Sprintf("JWT %s", response.Token),
	}

	status, _, err = utils.PostRequest(url, headers, payload)
	if status == types.ERROR {
		utils.PrintError(fmt.Sprintf("Error creating Docker Repository: %v", err))
		return types.ERROR, ""
	}

	return types.SUCCESS, fmt.Sprintf("%s/%s", username, name)
}

var createApiCmd = &cobra.Command{
	Use:   "api [name]",
	Short: "kubefs create api - create a new API resource",
	Long: `kubefs create api - create a new API resource
example: 
	kubefs create api my-api --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}
		
		if parseInfo(cmd, args, "api") == types.ERROR {
			return
		}

		var commands []string
		var up_local string

		if resourceFramework == "fast" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && python3 -m venv venv && source venv/bin/activate && pip install \"fastapi[standard]\" python-dotenv && pip freeze > requirements.txt && deactivate && touch main.py", resourceName),
				fmt.Sprintf("cd %s && echo 'from fastapi import FastAPI\napp = FastAPI()\n#KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n@app.get(\"/health\")\nasync def root():\n\treturn {\"status\": \"ok\"}' > main.py", resourceName),
			}

			up_local = fmt.Sprintf("source venv/bin/activate && uvicorn main:app --reload --port %v", resourcePort)
		}else if resourceFramework == "nest" {
			commands = []string{
				fmt.Sprintf("npx -p @nestjs/cli nest new %s -g -p npm", resourceName),
				fmt.Sprintf("cd %s/src/ && head -n 11 app.controller.ts > temp && mv temp app.controller.ts && echo '\t//KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n\t@Get(\"/health\")\n\tgetHealth(): string {\n\t\t return \"ok\";\n\t}\n}' >> app.controller.ts", resourceName),
			}

			up_local = fmt.Sprintf("PORT=%v npm run start:debug", resourcePort)
		}else{
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && go mod init %s && go get -u github.com/gorilla/mux", resourceName, resourceName),
				fmt.Sprintf("cd %s && echo 'package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n\t\"github.com/gorilla/mux\"\n)\n\nfunc main() {\n\tr := mux.NewRouter()\n\t//KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n\tr.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"ok\")\n\t})\n\tfmt.Println(\"Listening on Port %v\")\n\thttp.ListenAndServe(\":%v\", r)\n}' > main.go", resourceName, resourcePort, resourcePort),
			}

			up_local = fmt.Sprintf("go run main.go")
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}

		var dockerRepo string
		_, dockerRepo = createDockerRepo(resourceName)
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "api", Framework:resourceFramework, UpLocal: up_local, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), DockerRepo: dockerRepo, ClusterHost: fmt.Sprintf("http://%s-deploy.%s.svc.cluster.local", resourceName, resourceName)})
		
		err := utils.WriteManifest(&utils.ManifestData)
		if err == types.ERROR {
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
	kubefs create frontend my-frontend --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		if parseInfo(cmd, args, "frontend") == types.ERROR {
			return
		}

		var commands []string
		var start_command string

		if resourceFramework == "next" {
			commands = []string{
				fmt.Sprintf("npx create-next-app@latest %s --yes", resourceName),
				fmt.Sprintf("cd %s && rm -rf .git", resourceName),
			}

			start_command = fmt.Sprintf("next dev --turbopack --port %v", resourcePort)
		}else if resourceFramework == "remix" {
			commands = []string{
				fmt.Sprintf("npx create-remix@latest %s --no-git-init --yes", resourceName),
			}

			start_command = fmt.Sprintf("remix vite:dev --port %v", resourcePort)
		}else{
			commands = []string{
				fmt.Sprintf("npx sv create --template minimal --types ts --no-add-ons --no-install %s", resourceName),
				fmt.Sprintf("cd %s && npm i", resourceName),
			}
			start_command = fmt.Sprintf("vite dev --port %v", resourcePort)
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}

		err, packageJson := utils.ReadJson(fmt.Sprintf("%s/package.json", resourceName))
		if err == types.ERROR {
			utils.PrintError("Error reading package.json")
			return
		}

		packageJson["scripts"].(map[string]interface{})["dev"] = start_command

		err = utils.WriteJson(packageJson, fmt.Sprintf("%s/package.json", resourceName))
		if err == types.ERROR {
			utils.PrintError("Error writing package.json")
			return
		}

		var dockerRepo string
		_, dockerRepo = createDockerRepo(resourceName)
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "frontend", Framework:resourceFramework, UpLocal: "npm run dev", LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), DockerRepo: dockerRepo, ClusterHost: fmt.Sprintf("http://%s-deploy.%s.svc.cluster.local", resourceName, resourceName)})
		
		err = utils.WriteManifest(&utils.ManifestData)
		if err == types.ERROR {
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
	kubefs create database mydb --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}
		
		if parseInfo(cmd, args, "database") == types.ERROR {
			return
		}

		var input string
		fmt.Print("Enter the password for the database: ")
		fmt.Scanln(&input)
		password := strings.TrimSpace(input)

		var commands []string
		dockerRepo := fmt.Sprintf("bitnami/%s", resourceFramework)
		
		commands = []string{
			fmt.Sprintf("mkdir %s", resourceName),
		}

		var clusterHost string
		if resourceFramework == "cassandra" {
			clusterHost = fmt.Sprintf("http://%s-%s.%s.svc.cluster.local:%v", resourceName, resourceFramework, resourceName, resourcePort)		
		}else{
			clusterHost = fmt.Sprintf("http://%s-redis-master.%s.svc.cluster.local:%v", resourceName, resourceName, resourcePort)	
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "database", Framework:resourceFramework, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("http://%s:%v", resourceName, resourcePort), DockerRepo: dockerRepo, ClusterHost: clusterHost, DbPassword: password})

		fileErr := utils.WriteManifest(&utils.ManifestData)
		if fileErr == types.ERROR {
			return
		}
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
	createDbCmd.Flags().StringP("framework", "f", "cassandra", "Type of database to use [cassandra | redis | elasticsearch]")

	createCmd.PersistentFlags().IntP("port", "p", 3000, "Specific port to be used")
}
