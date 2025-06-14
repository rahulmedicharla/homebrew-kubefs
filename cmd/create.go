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
	"github.com/goodhosts/hostsfile"
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
	var desc string
	err := utils.ReadInput("Enter resource description: ", &desc)
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

var createHostEntry = &cobra.Command{
	Use:   "host-entry [ip-address] [host-domain]",
	Short: "kubefs create host-entry - add new host entry to the hosts file",
	Long: `kubefs create host-entry - add new host entry to the hosts file
example: 
	kubefs create host-entry <ip-address> <host-domain>,
	`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil{
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}
		
		ipAddress := args[0]
		hostDomain := args[1]

		hosts, err := hostsfile.NewHosts()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating hosts file. %v", err.Error()))
			return
		}

		err = hosts.Add(ipAddress, hostDomain)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error adding host entry. %v", err.Error()))
			return
		}

		err = hosts.Flush()
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error flushing hosts file. %v", err.Error()))
			return
		}
		utils.PrintSuccess(fmt.Sprintf("Successfully added host entry %s -> %s", hostDomain, ipAddress))

	},
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
				fmt.Sprintf("cd %s && go mod init %s && go get -u github.com/gin-gonic/gin", resourceName, resourceName),
				fmt.Sprintf("cd %s && echo 'package main\n\nimport (\n\t\"log\"\n\t\"net/http\"\n\t\"github.com/gin-gonic/gin\"\n)\n\nfunc main() {\n\tr := gin.Default()\n\t//KEEP THIS PATH BELOW, IT ACTS AS A READINESS CHECK IN KUBERNETES\n\tr.GET(\"/health\", func(c *gin.Context) {\n\t\tc.JSON(http.StatusOK, gin.H{\n\t\t\t\"status\": \"ok\",\n\t\t})\n\t})\n\tlog.Println(\"Listening on Port %v\")\n\thttp.ListenAndServe(\":%v\", r)\n}' > main.go", resourceName, resourcePort, resourcePort),
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
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "api", 
			Framework:resourceFramework, 
			AttachCommand : map[string]string{
				"docker": fmt.Sprintf("docker exec -it %s-%s-1 sh", utils.ManifestData.KubefsName, resourceName),
				"kubernetes": fmt.Sprintf("kubectl exec -it svc/%s-deploy -n %s -- sh", resourceName, resourceName),
			},
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

		var hostDomain string

		err := utils.ReadInput("Enter the host domain the ingresss should accept: ", &hostDomain)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
			return
		}

		// add hostDomain to host file
		err = utils.AddHost("127.0.0.1", hostDomain)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error adding host to host file. %v", err.Error()))
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

		err = utils.RunMultipleCommands(commands, true, true)
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
		}

		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "frontend", 
			Framework:resourceFramework, 
			AttachCommand : map[string]string{
				"docker": fmt.Sprintf("docker exec -it %s-%s-1 sh", utils.ManifestData.KubefsName, resourceName),
				"kubernetes": fmt.Sprintf("kubectl exec -it svc/%s-deploy -n %s -- sh", resourceName, resourceName),
			},
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

		var password string
		var persistence int

		err := utils.ReadInput("Enter a password for the database: ", &password)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
			return
		}

		err = utils.ReadInput("How many Gigabytes of persistence on each pod: (ex 1): ", &persistence)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
			return
		}

		dockerRepo := fmt.Sprintf("bitnami/%s", resourceFramework)
		
		var clusterHost string
		var clusterHostRead string
		var dockerHost string
		var localHost string
		var defaultDatabase string = "0"
		var user string = "default"
		var attachCommand map[string]string
		if resourceFramework == "postgresql" {
			err = utils.ReadInput("Enter the database to be initialized on init: ", &defaultDatabase)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
				return 
			}
		
			err = utils.ReadInput("Enter a username for the database: ", &user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Unexpected error reading input. %v", err.Error()))
				return
			}
			
			localHost = fmt.Sprintf("postgresql://%s:%s@localhost:%v/%s?sslmode=disable", user, password, resourcePort, defaultDatabase)
			dockerHost = fmt.Sprintf("postgresql://%s:%s@%s:%v/%s?sslmode=disable", user, password, resourceName, resourcePort, defaultDatabase)
			clusterHost = fmt.Sprintf("postgresql://%s:%s@%s-postgresql-primary.svc.cluster.local:80/%s?sslmode=disable", user, password, resourceName, defaultDatabase)
			clusterHostRead = fmt.Sprintf("postgresql://%s:%s@%s-postgresql-read.svc.cluster.local:80/%s?sslmode=disable", user, password, resourceName, defaultDatabase)		

			attachCommand = map[string]string{
				"docker": fmt.Sprintf("docker exec -it %s-%s-1 sh -c 'PGPASSWORD=%s psql -U %s -p %v -d %s'", utils.ManifestData.KubefsName, resourceName, password, user, resourcePort, defaultDatabase),
				"kubernetes": fmt.Sprintf("kubectl exec -it svc/%s-postgresql-primary -n %s -- env PGPASSWORD=%s psql -U %s -d %s", resourceName, resourceName, password, user, defaultDatabase),
			}

		}else{
			localHost = fmt.Sprintf("redis://default:%s@localhost:%v", password, resourcePort)
			dockerHost = fmt.Sprintf("redis://default:%s@%s:%v", password, resourceName, resourcePort)
			clusterHost = fmt.Sprintf("redis://default:%s@%s-redis-master.svc.cluster.local:80", password, resourceName)
			clusterHostRead = fmt.Sprintf("redis://default:%s@%s-redis-replicas.svc.cluster.local:80", password, resourceName)

			attachCommand = map[string]string{
				"docker": fmt.Sprintf("docker exec -it %s-%s-1 redis-cli -p %v -a %s", utils.ManifestData.KubefsName, resourceName, resourcePort, password),
				"kubernetes": fmt.Sprintf("kubectl exec -it svc/%s-redis-master -n %s -- redis-cli -a %s", resourceName, resourceName, password),
			}
		}

		err = utils.RunCommand(fmt.Sprintf("mkdir %s", resourceName), true, true)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Unexpected error creating resource. %v", err.Error()))
			return
		}
		
		utils.ManifestData.Resources = append(utils.ManifestData.Resources, types.Resource{
			Name: resourceName, 
			Port: resourcePort, 
			Type: "database", 
			Framework:resourceFramework, 
			AttachCommand: attachCommand,
			LocalHost: localHost, 
			DockerHost: dockerHost,
			DockerRepo: dockerRepo, 
			ClusterHost: clusterHost, 
			ClusterHostRead: clusterHostRead,
			Opts: map[string]string{
				"user": user,
				"password": password,
				"default-database": defaultDatabase,
				"persistence": fmt.Sprintf("%vGi", persistence),
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
	createApiCmd.Flags().StringP("framework", "f", "fast", "Framework to use for API [fast | nest | gin]")
	createFrontendCmd.Flags().StringP("framework", "f", "next", "Framework to use for Frontend [next | remix | sveltekit]")
	createDbCmd.Flags().StringP("framework", "f", "postgresql", "Type of database to use [postgresql | redis]")

	createCmd.AddCommand(createHostEntry)

	createCmd.PersistentFlags().IntP("port", "p", 3000, "Specific port to be used")
}
