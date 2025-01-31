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

// createCmd represents the create command

var resourcePort int
var resourceFramework string
var resourceName string

var createCmd = &cobra.Command{
	Use:   "create [command]",
	Short: "kubefs create - easily create backend, frontend, & db resources to be used within your application",
	Long: `kubefs create - easily create backend, frontend, & db resources to be used within your application`,
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
	if err != nil || port == 6000 {
		utils.PrintError(fmt.Sprintf(" Invalid port. Port 6000 reserved for kubefs: %v", err))
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
	fmt.Scanln(&input)
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
	Long: `kubefs create api - create a new API resource`,
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
		var framework string

		if resourceFramework == "fast" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && python3 -m venv venv && source venv/bin/activate && pip install \"fastapi[standard]\" && pip freeze > requirements.txt && deactivate && touch main.py", resourceName),
				fmt.Sprintf("cd %s && echo 'from fastapi import FastAPI\napp = FastAPI()\n@app.get(\"/\")\nasync def root():\n\treturn {\"message\": \"Hello World\"}' > main.py", resourceName),
			}

			up_local = fmt.Sprintf("source venv/bin/activate && uvicorn main:app --port %v", resourcePort)
			framework = "fast"			
		}else if resourceFramework == "koa" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && npm init -y && npm i koa nodemon", resourceName),
				fmt.Sprintf("cd %s && echo '\"use strict\";\nconst Koa = require(\"koa\");\nconst app = new Koa();\n\napp.use(ctx => {\n\tctx.body = \"Hello World\";\n});\n\napp.listen(%v);' > index.js", resourceName, resourcePort),
			}

			up_local = fmt.Sprintf("cd %s && npx nodemon index.js", resourceName)
			framework = "koa"
		}else{
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && go mod init %s && go get -u github.com/gorilla/mux", resourceName, resourceName),
				fmt.Sprintf("cd %s && echo 'package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n\t\"github.com/gorilla/mux\"\n)\n\nfunc main() {\n\tr := mux.NewRouter()\n\tr.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"Hello World\")\n\t})\n\tfmt.Println(\"Listening on Port %v\")\n\thttp.ListenAndServe(\":%v\", r)\n}' > main.go", resourceName, resourcePort, resourcePort),
			}

			up_local = fmt.Sprintf("cd %s && go run main.go", resourceName)
			framework = "go"
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
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "api", Framework:framework, UpLocal: up_local, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("%s-api-1:%v", resourceName, resourcePort), DockerRepo: dockerRepo, ClusterHost: fmt.Sprintf("%s-deployment.%s.svc.cluster.local:%v", resourceName, resourceName, resourcePort)})
		
		err := utils.WriteManifest(&utils.ManifestData)
		if err == types.ERROR {
			return
		}
		utils.PrintSuccess(fmt.Sprintf("Successfully created API %s on port %v using the %s framework", resourceName, resourcePort, resourceFramework))
	},
}

var createFrontendCmd = &cobra.Command{
	Use:   "frontend [name]",
	Short: "kubefs create frontend - create a new Frontend resource",
	Long: `kubefs create frontend - create a new Frontend resource`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		if parseInfo(cmd, args, "frontend") == types.ERROR {
			return
		}

		var commands []string
		var up_local string
		var framework string

		if resourceFramework == "react" {
			commands = []string{
				fmt.Sprintf("npx -p yarn yarn create react-app %s --no-git --template typescript --silent", resourceName),
				fmt.Sprintf("cd %s && rm -rf .git; echo ''", resourceName),
			}

			up_local = fmt.Sprintf("cd %s && export PORT=%v && npm start", resourceName, resourcePort)
			framework = "react"			
		}else if resourceFramework == "angular" {
			commands = []string{
				fmt.Sprintf("npx -p @angular/cli ng new %s --defaults --skip-git", resourceName),
			}

			up_local = fmt.Sprintf("cd %s && npx -p @angular/cli ng serve --port %v", resourceName, resourcePort)
			framework = "angular"
		}else{
			commands = []string{
				fmt.Sprintf("npm create vue@latest %s -- --typescript", resourceName),
				fmt.Sprintf("cd %s && npm install && rm vite.config.ts", resourceName),
			}

			up_local = fmt.Sprintf("cd %s && npm run dev -- --port %v", resourceName, resourcePort)
			framework = "vue"
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}

		if resourceFramework == "react"{
			err, packageJson := utils.ReadJson(fmt.Sprintf("%s/package.json", resourceName))
			if err == types.ERROR {
				utils.PrintError("Error reading package.json")
				return
			}
			packageJson["proxy"] = fmt.Sprintf("http://localhost:6000")
			err = utils.WriteJson(packageJson, fmt.Sprintf("%s/package.json", resourceName))
			if err == types.ERROR {
				utils.PrintError("Error writing package.json")
				return
			}
		}else if resourceFramework == "angular" {
			writeErr := os.WriteFile(fmt.Sprintf("%s/proxy.conf.json", resourceName), []byte("{\n    \"/env\": {\n      \"target\": \"http://localhost:6000\",\n      \"secure\": false\n    }\n  }"), 0644)
			if writeErr != nil {
				utils.PrintError(fmt.Sprintf("Error writing proxy.conf.json: %v", writeErr))
				return
			}
			err, angularJson := utils.ReadJson(fmt.Sprintf("%s/angular.json", resourceName))
			if err == types.ERROR {
				utils.PrintError("Error reading angular.json")
				return
			}
			proxyInfo := map[string]interface{}{
				"proxyConfig": "proxy.conf.json",
			}

			angularJson["projects"].(map[string]interface{})[fmt.Sprintf("%s", resourceName)].(map[string]interface{})["architect"].(map[string]interface{})["serve"].(map[string]interface{})["options"] = proxyInfo
			// projects := angularJson["projects"].(map[string]interface{})
			// project := projects[resourceName].(map[string]interface{})
			// architect := project["architect"].(map[string]interface{})
			// serve := architect["serve"].(map[string]interface{})
			// options := serve["options"].(map[string]interface{})
			// options["proxyConfig"] = fmt.Sprintf("proxy.conf.json")
			err = utils.WriteJson(angularJson, fmt.Sprintf("%s/angular.json", resourceName))
			if err == types.ERROR {
				utils.PrintError("Error writing angular.json")
				return
			}
		}else{
			writeErr := os.WriteFile(fmt.Sprintf("%s/vite.config.ts", resourceName), []byte("import { fileURLToPath, URL } from 'node:url'\n\nimport { defineConfig } from 'vite'\nimport vue from '@vitejs/plugin-vue'\n\n// https://vitejs.dev/config/\nexport default defineConfig({\n  plugins: [\n    vue(),\n  ],\n  server: {\n    proxy: {\n      \"/env\": {\n        target: \"http://localhost:6000\",\n        changeOrigin: true,\n        rewrite: (path) => path.replace(/^\\/env/, '/env'),\n      }\n    }\n  },\n  resolve: {\n    alias: {\n      '@': fileURLToPath(new URL('./src', import.meta.url))\n    }\n  }\n})"), 0644)
			if writeErr != nil {
				utils.PrintError(fmt.Sprintf("Error writing vite.config.ts: %v", writeErr))
				return
			}
		}

		var dockerRepo string
		_, dockerRepo = createDockerRepo(resourceName)
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "frontend", Framework:framework, UpLocal: up_local, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("%s-frontend-1:%v", resourceName, resourcePort), DockerRepo: dockerRepo, ClusterHost: fmt.Sprintf("%s-deployment.%s.svc.cluster.local:%v", resourceName, resourceName, resourcePort)})
		
		err := utils.WriteManifest(&utils.ManifestData)
		if err == types.ERROR {
			return
		}
		utils.PrintSuccess(fmt.Sprintf("Successfully created frontend %s on port %v using the %s framework", resourceName, resourcePort, resourceFramework))
	},
}

var createDbCmd = &cobra.Command{
	Use:   "database [name]",
	Short: "kubefs create database - create a new database resource",
	Long: `kubefs create database - create a new database resource`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}
		
		if parseInfo(cmd, args, "database") == types.ERROR {
			return
		}

		var commands []string
		var up_local string
		var framework string

		if resourceFramework == "cassandra" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
			}
			framework = "cassandra"		
		}else{
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
			}
			framework = "mongodb"
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "database", Framework:framework, UpLocal: up_local, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort), DockerHost: fmt.Sprintf("%s-container-1:%v", resourceName, resourcePort), ClusterHost: fmt.Sprintf("%s-deployment.%s.svc.cluster.local:%v", resourceName, resourceName, resourcePort)})
		
		err := utils.WriteManifest(&utils.ManifestData)
		if err == types.ERROR {
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
	createApiCmd.Flags().StringP("framework", "f", "fast", "Framework to use for API [fast | koa | go]")
	createFrontendCmd.Flags().StringP("framework", "f", "react", "Framework to use for Frontend [react | vue | angular]")
	createDbCmd.Flags().StringP("framework", "f", "cassandra", "Type of database to use [cassandra | mongodb]")

	createCmd.PersistentFlags().IntP("port", "p", 3000, "Specific port to be used")
}
