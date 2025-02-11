/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"strings"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy [command]",
	Short: "kubefs deploy - create helm charts & deploy the build targets onto the cluster",
	Long: `kubefs deploy - create helm charts & deploy the build targets onto the cluster
example:
	kubefs deploy all --flags,
	kubefs deploy resource <frontend>,<api>,<database> --flags,
	kubefs deploy resource <frontend> --flags,`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func deployAddon(addon *types.Addon, onlyHelmify bool, onlyDeploy bool) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", addon.DockerRepo), true, true)
	if err != nil {
		return err
	}

	if !onlyDeploy {
		// helmify
		var valuesYaml map[string]interface{}
		if addon.Name == "oauth2"{
			if err = utils.DownloadZip(types.OAUTH2CHART, "addons/oauth2"); err != nil {
				return err
			}

			valuesYaml = *utils.GetHelmChart(addon.DockerRepo, addon.Name, "ClusterIP", addon.Port, false, "", "/health", 1)

			env := valuesYaml["env"].([]interface{})
			var allowedOrigins []string
			for _, n := range addon.Dependencies {
				attachedResource, err := utils.GetResourceFromName(n)
				if err != nil {
					return err
				}

				allowedOrigins = append(allowedOrigins, attachedResource.ClusterHost)
			}
			
			env = append(env, map[string]interface{}{
				"name": "ALLOWED_ORIGINS", 
				"value": strings.Join(allowedOrigins, ","),
			}, map[string]interface{}{
				"name": "PORT",
				"value": fmt.Sprintf("%v", addon.Port),
			}, map[string]interface{}{
				"name": "GIN_MODE",
				"value": "release",
			})
			valuesYaml["env"] = env

			valuesYaml["secrets"] = []interface{}{
				map[string]interface{}{
					"name": "public_key.pem",
					"value": "files/public_key.pem",
				},
				map[string]interface{}{
					"name": "private_key.pem",
					"value": "files/private_key.pem",
				},
			}

			err = utils.RunCommand(fmt.Sprintf("mkdir -p addons/oauth2/deploy/files && cp addons/oauth2/public_key.pem addons/oauth2/deploy/files && cp addons/oauth2/private_key.pem addons/oauth2/deploy/files"), true, true)
			if err != nil {
				return err
			}

			valuesYaml["volumes"] = []interface{}{
				map[string]interface{}{
					"name": "store",
					"emptyDir": map[string]interface{}{},
				},
				map[string]interface{}{
					"name": "keys",
					"secret": map[string]string{
						"secretName": "oauth2-deploy-secret",
					},
				},
			}

			valuesYaml["volumeMounts"] = []interface{}{
				map[string]string{
					"name": "store",
					"mountPath": "/app/store",
				},
				map[string]string{
					"name": "keys",
					"mountPath": "/etc/ssl/private/private_key.pem",
					"subPath": "private_key.pem",
				},
				map[string]string{
					"name": "keys",
					"mountPath": "/etc/ssl/public/public_key.pem",
					"subPath": "public_key.pem",
				},
			}

			err = utils.WriteYaml(&valuesYaml, "addons/oauth2/deploy/values.yaml")
			if err != nil {
				return err
			}
		}
	}
	if !onlyHelmify {
		// deploy
		if addon.Name == "oauth2"{
			err = utils.RunCommand(fmt.Sprintf("helm upgrade --install oauth2 addons/oauth2/deploy"), true, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deployUnique(resource *types.Resource, onlyHelmify bool, onlyDeploy bool) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", resource.DockerRepo), true, true)
	if err != nil {
		return err
	}

	if !onlyDeploy {
		// helmify
		if resource.Type == "database"{
			// database
			var cmds []string
			if resource.Framework == "cassandra"{
				cmds = append(cmds, 
					fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/cassandra --untar && mv cassandra deploy)", resource.Name),
					fmt.Sprintf("echo 'connect to cassandra by exec into it and cqlsh [host] [port] -u cassandra -p [password]' > %s/deploy/templates/NOTES.txt", resource.Name),
				)
			}else{
				cmds = append(cmds, 
					fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/redis --untar && mv redis deploy)", resource.Name),
					fmt.Sprintf("echo 'connect to redis by exec into it and redis-cli -h [host] -p [port] -a [password]' > %s/deploy/templates/NOTES.txt", resource.Name),
				)
			}

			err = utils.RunMultipleCommands(cmds, true, true)
			if err != nil {
				return err
			}

		}else{
			// api or frontend
			if err = utils.DownloadZip(types.HELMCHART, resource.Name); err != nil {
				return err
			}

			var valuesYaml map[string]interface{}
			if resource.Type == "api"{
				// api
				valuesYaml = *utils.GetHelmChart(resource.DockerRepo, resource.Name, "ClusterIP", resource.Port, false, "", "/health", 3)
			}else{
				// frontend
				valuesYaml = *utils.GetHelmChart(resource.DockerRepo, resource.Name, "NodePort", resource.Port, true, resource.UrlHost, "/", 3)
			}

			env := valuesYaml["env"].([]interface{})
			for _, r := range utils.ManifestData.Resources {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
			}

			for _, a := range utils.ManifestData.Addons {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", a.Name), "value": a.ClusterHost})
			}
			valuesYaml["env"] = env
		
			envData, err := utils.ReadEnv(fmt.Sprintf("%s/.env", resource.Name))			
			if err == nil {
				secrets := valuesYaml["secrets"].([]interface{})
				for _,line := range envData {
					secrets = append(secrets, map[string]interface{}{"name": strings.Split(line, "=")[0], "value": strings.Split(line, "=")[1], "secretRef": fmt.Sprintf("%s-deploy-secret", resource.Name)})
				}
				valuesYaml["secrets"] = secrets
			}

			err = utils.WriteYaml(&valuesYaml, fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err != nil {
				return err
			}
		}
	}

	if !onlyHelmify {
		// deploy
		var cmd string
		if resource.Type == "database"{
			if resource.Framework == "cassandra"{
				cmd = fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set persistence.size=1Gi --set containerPorts.cql=%v --set service.ports.cql=80 --set dbUser.password=%s --set cluster.datacenter=datacenter1 --set replicaCount=3 --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.DbPassword, resource.Name)
			}else{
				cmd = fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set persistence.size=1Gi --set master.containerPorts.redis=%v --set master.service.ports.redis=80 --set auth.password=%s --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.DbPassword, resource.Name)
			}
		}else{
			cmd = fmt.Sprintf("helm upgrade --install %s %s/deploy", resource.Name, resource.Name)
		}

		err = utils.RunCommand(cmd, true, true)
		if err != nil {
			return err
		}
	}

	return nil

}

var deployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources",
	Long: `kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources
example: 
	kubefs deploy all --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")

		var errors []string
		var successes []string
		var hosts []string

        utils.PrintWarning("Deploying all resources & addons")

        for _, resource := range utils.ManifestData.Resources {
			err := deployUnique(&resource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s. %v", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}

			successes = append(successes, resource.Name)
			if resource.Type == "frontend" {
				if resource.UrlHost == "" {
					hosts = append(hosts, "*")
				}else{
					hosts = append(hosts, resource.UrlHost)
				}
			}
        }

		for _, addon := range utils.ManifestData.Addons {
			err := deployAddon(&addon, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s. %v", addon.Name, err.Error()))
				errors = append(errors, addon.Name)
				continue
			}
			successes = append(successes, addon.Name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error deploying resource %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", successes))
		}

		if len(hosts) > 0 {
			utils.PrintWarning(fmt.Sprintf("Frontend resources are available at %v", hosts))
		}
	},
}

var deployResourceCmd = &cobra.Command{
	Use:   "resource [name, ...]",
	Short: "kubefs deploy resource [name, ...] - create helm charts & deploy the build targets onto the cluster for listed resource",
	Long: `kubefs deploy resource [name, ...] - create helm charts & deploy the build targets onto the cluster for listed resource
example: 
	kubefs deploy resource <frontend>,<api>,<database> --flags,
	kubefs deploy resource <frontend> --flags
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

		names := strings.Split(args[0], ",")

		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")
		
		addons, _ := cmd.Flags().GetString("with-addons")
		addonList := strings.Split(addons, ",")

		var successes []string
		var errors []string
		var hosts []string

		utils.PrintWarning(fmt.Sprintf("Deploying resource %v", names))
		utils.PrintWarning(fmt.Sprintf("Including addons %v", addonList))

		for _, name := range names {
			var resource *types.Resource
			resource, err := utils.GetResourceFromName(name)

			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = deployUnique(resource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
			}

			successes = append(successes, name)
			if resource.Type == "frontend" {
				if resource.UrlHost == "" {
					hosts = append(hosts, "*")
				}else{
					hosts = append(hosts, resource.UrlHost)
				}
			}

		}

		for _, addon := range addonList {
			if addon == ""{
				continue
			}
			var addonResource *types.Addon
			addonResource, err := utils.GetAddonFromName(addon)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, addon)
				continue
			}

			err = deployAddon(addonResource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s. %v", addon, err.Error()))
				errors = append(errors, addon)
				continue
			}

			successes = append(successes, addon)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error deploying resource %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", successes))
		}

		if len(hosts) > 0 {
			utils.PrintWarning(fmt.Sprintf("Frontend resources are available at %v", hosts))
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployAllCmd)
	deployCmd.AddCommand(deployResourceCmd)

	deployCmd.PersistentFlags().BoolP("only-helmify", "w", false, "only helmify the resources")
	deployCmd.PersistentFlags().BoolP("only-deploy", "d", false, "only deploy the resources")
	
	deployResourceCmd.Flags().StringP("with-addons", "a", "", "addons to be included in deployment (comma separated)")

}
