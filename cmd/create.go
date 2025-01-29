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
)

// createCmd represents the create command

var ManifestData types.Project
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
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error reading port: %v", err))
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
	if !utils.Contains(allowableFrameworks, framework) {
		utils.PrintError(fmt.Sprintf("Invalid framework: %s. Allowed frameworks are: %v", framework, allowableFrameworks))
		return types.ERROR
	}

	for _, resource := range ManifestData.Resources {
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

var createApiCmd = &cobra.Command{
	Use:   "api [name]",
	Short: "kubefs create api - create a new API resource",
	Long: `kubefs create api - create a new API resource`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ValidateProject() == types.ERROR {
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
				fmt.Sprintf("cd %s && python3 -m venv venv && source venv/bin/activate && pip install \"fastapi[standard]\" && deactivate && touch main.py", resourceName),
				fmt.Sprintf("cd %s && echo 'from fastapi import FastAPI\napp = FastAPI()\n@app.get(\"/\")\nasync def root():\n\treturn {\"message\": \"Hello World\"}' > main.py", resourceName),
			}

			up_local = fmt.Sprintf("source venv/bin/activate && uvicorn main:app --port %v", resourcePort)
			
		}else if resourceFramework == "koa" {
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && npm init -y && npm i koa nodemon", resourceName),
				fmt.Sprintf("cd %s && echo '\"use strict\";\nconst Koa = require(\"koa\");\nconst app = new Koa();\n\napp.use(ctx => {\n\tctx.body = \"Hello World\";\n});\n\napp.listen(%v);' > index.js", resourceName, resourcePort),
			}

			up_local = fmt.Sprintf("cd %s && npx nodemon index.js", resourceName)
		}else{
			commands = []string{
				fmt.Sprintf("mkdir %s", resourceName),
				fmt.Sprintf("cd %s && go mod init %s && go get -u github.com/gorilla/mux", resourceName, resourceName),
				fmt.Sprintf("cd %s && echo 'package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n\t\"github.com/gorilla/mux\"\n)\n\nfunc main() {\n\tr := mux.NewRouter()\n\tr.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n\t\tfmt.Fprintf(w, \"Hello World\")\n\t})\n\tfmt.Println(\"Listening on Port %v\")\n\thttp.ListenAndServe(\":%v\", r)\n}' > main.go", resourceName, resourcePort, resourcePort),
			}

			up_local = fmt.Sprintf("cd %s && go run main.go", resourceName)
		}

		for _, command := range commands {
			cmd := exec.Command("sh", "-c", command)
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected Error %v", err))
				return
			}
		}
		
		ManifestData.Resources = append(ManifestData.Resources, types.Resource{Name: resourceName, Port: resourcePort, Type: "api", UpLocal: up_local, LocalHost: fmt.Sprintf("http://localhost:%v", resourcePort)})
		
		err := utils.WriteManifest(&ManifestData)
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
		if utils.ValidateProject() == types.ERROR {
			return
		}

		if parseInfo(cmd, args, "frontend") == types.ERROR {
			return
		}
	},
}

var createDbCmd = &cobra.Command{
	Use:   "database [name]",
	Short: "kubefs create database - create a new database resource",
	Long: `kubefs create database - create a new database resource`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ValidateProject() == types.ERROR {
			return
		}

		if parseInfo(cmd, args, "database") == types.ERROR {
			return
		}
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

	utils.ReadManifest(&ManifestData)
}
