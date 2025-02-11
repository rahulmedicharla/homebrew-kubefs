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
	"os"
	"net/http"
	"io"
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

func downloadZip(url string, name string) int {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("(cd %s && rm -rf deploy)", name))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdErr := cmd.Run()
	if cmdErr != nil {
		utils.PrintError(fmt.Sprintf("Error removing old helm charts: %v", cmdErr))
		return types.ERROR
	}

	resp, err := http.Get(url)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error downloading helm chart: %v", err))
		return types.ERROR
	}

	defer resp.Body.Close()

	out, err := os.Create(fmt.Sprintf("%s/helm.zip", name))
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error creating helm chart: %v", err))
		return types.ERROR
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error copying helm chart: %v", err))
		return types.ERROR
	}

	cmd = exec.Command("sh", "-c", fmt.Sprintf("(cd %s && unzip helm.zip -d deploy && rm -rf helm.zip deploy/__MACOSX && echo '')", name))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdErr = cmd.Run()
	if cmdErr != nil {
		utils.PrintError(fmt.Sprintf("Error unzipping helm chart: %v", cmdErr))
		return types.ERROR
	}

	return types.SUCCESS
}

func deployAddon(addon *types.Addon, onlyHelmify bool, onlyDeploy bool) int {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker pull %s", addon.DockerRepo))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Docker image %s not found. Run 'kubefs compile' to set docker images", addon.Name))
		return types.ERROR
	}

	if !onlyDeploy {
		// helmify

		if addon.Name == "oauth2"{
			err := downloadZip(types.OAUTH2CHART, "addons/oauth2")
			if err == types.ERROR {
				return types.ERROR
			}

			var defaultYaml = types.GetHelmChart(addon.DockerRepo, addon.Name, "ClusterIP", addon.Port, "false", "", "/health", fmt.Sprintf("%v", addon.Port), 1)
			
			fileWriteErr := os.WriteFile("addons/oauth2/deploy/values.yaml", []byte(defaultYaml), 0644)

			if fileWriteErr != nil {
				utils.PrintError(fmt.Sprintf("Error writing values.yaml: %v", fileWriteErr))
				return types.ERROR
			}

			err, valuesYaml := utils.ReadYaml("addons/oauth2/deploy/values.yaml")
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error reading values.yaml: %v", err))
				return types.ERROR
			}

			env := valuesYaml["env"].([]interface{})
			allowedOrigins := ""
			for _, n := range addon.Dependencies {
				attachedResource := utils.GetResourceFromName(n)
				if attachedResource == nil {
					utils.PrintError(fmt.Sprintf("Resource %s not found", n))
					continue
				}
				if allowedOrigins == "" {
					allowedOrigins = fmt.Sprintf("%s", attachedResource.ClusterHost)
				}else{
					allowedOrigins += fmt.Sprintf(",%s", attachedResource.ClusterHost)
				}
			}
			env = append(env, map[string]interface{}{
				"name": "ALLOWED_ORIGINS", 
				"value": allowedOrigins,
			}, map[string]interface{}{
				"name": "PORT",
				"value": fmt.Sprintf("%v", addon.Port),
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

			cmd = exec.Command("sh", "-c", fmt.Sprintf("mkdir -p addons/oauth2/deploy/files && cp addons/oauth2/public_key.pem addons/oauth2/deploy/files && cp addons/oauth2/private_key.pem addons/oauth2/deploy/files"))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error copying files: %v", cmdErr))
				return types.ERROR
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
			if err == types.ERROR {
				return types.ERROR
			}
		}
	}
	if !onlyHelmify {
		// deploy
		if addon.Name == "oauth2"{
			cmd := exec.Command("sh", "-c", "helm upgrade --install oauth2 ./addons/oauth2/deploy")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s: %v", addon.Name, cmdErr))
				return types.ERROR
			}
		}
	}

	return types.SUCCESS
}

func deployUnique(resource *types.Resource, onlyHelmify bool, onlyDeploy bool) int {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker pull %s", resource.DockerRepo))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Docker image %s not found. Run 'kubefs compile' to set docker images", resource.Name))
		return types.ERROR
	}

	if !onlyDeploy {
		// helmify

		if resource.Type == "database"{
			// database
			var cmd *exec.Cmd

			if resource.Framework == "cassandra"{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/cassandra --untar && mv cassandra deploy)", resource.Name))
			}else{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/redis --untar && mv redis deploy)", resource.Name))
			}

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error pulling helm chart: %v", cmdErr))
				return types.ERROR
			}

		}else{
			// api or frontend
			err := downloadZip(types.HELMCHART, resource.Name)
			if err == types.ERROR {
				return types.ERROR
			}

			var defaultYaml string
			if resource.Type == "api"{
				// api
				defaultYaml = types.GetHelmChart(resource.DockerRepo, resource.Name, "ClusterIP", resource.Port, "false", "", "/health", "http", 3)
			}else{
				// frontend
				defaultYaml = types.GetHelmChart(resource.DockerRepo, resource.Name, "NodePort", resource.Port, "true", resource.UrlHost, "/", "http", 3)
			}

			fileWriteErr := os.WriteFile(fmt.Sprintf("%s/deploy/values.yaml", resource.Name), []byte(defaultYaml), 0644)

			if fileWriteErr != nil {
				utils.PrintError(fmt.Sprintf("Error writing values.yaml: %v", fileWriteErr))
				return types.ERROR
			}

			err, valuesYaml := utils.ReadYaml(fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error reading values.yaml: %v", err))
				return types.ERROR
			}
			env := valuesYaml["env"].([]interface{})
			for _, r := range utils.ManifestData.Resources {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
			}
			for _, a := range utils.ManifestData.Addons {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", a.Name), "value": a.ClusterHost})
			}
			valuesYaml["env"] = env
		
			envErr, envData := utils.ReadEnv(fmt.Sprintf("%s/.env", resource.Name))
			if envErr == types.SUCCESS {
				secrets := valuesYaml["secrets"].([]interface{})
				for _,line := range envData {
					secrets = append(secrets, map[string]interface{}{"name": strings.Split(line, "=")[0], "value": strings.Split(line, "=")[1], "secretRef": fmt.Sprintf("%s-deploy-secret", resource.Name)})
				}
				valuesYaml["secrets"] = secrets
			}

			err = utils.WriteYaml(&valuesYaml, fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}
		}
	}

	if !onlyHelmify {
		// deploy
		var cmd *exec.Cmd

		if resource.Type == "database"{
			if resource.Framework == "cassandra"{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set service.ports.cql=%v --set persistence.size=1Gi --set dbUser.password=%s --set cluster.datacenter=datacenter1 --set replicaCount=3 --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.DbPassword, resource.Name))
			}else{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set persistence.size=1Gi --set master.service.ports.redis=%v --set auth.password=%s --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.DbPassword, resource.Name))
			}
		}else{
			cmd = exec.Command("sh", "-c", fmt.Sprintf("helm upgrade --install %s %s/deploy", resource.Name, resource.Name))
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr := cmd.Run()
		if cmdErr != nil {
			utils.PrintError(fmt.Sprintf("Error deploying resource %s: %v", resource.Name, cmdErr))
			return types.ERROR
		}
	}

	return types.SUCCESS

}

var deployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources",
	Long: `kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources
example: 
	kubefs deploy all --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var errors []string
		var successes []string
		var hosts []string

        utils.PrintWarning("Deploying all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := deployUnique(&resource, onlyHelmify, onlyDeploy)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s", resource.Name))
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
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s", addon.Name))
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

		names := strings.Split(args[0], ",")
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}


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
			resource = utils.GetResourceFromName(name)

			if resource == nil {
				utils.PrintError(fmt.Sprintf("Resource %s not found", name))
				continue
			}

			err := deployUnique(resource, onlyHelmify, onlyDeploy)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s", name))
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
			var addonResource *types.Addon
			addonResource = utils.GetAddonFromName(addon)

			if addonResource == nil {
				utils.PrintError(fmt.Sprintf("Addon %s not found", addon))
				continue
			}

			err := deployAddon(addonResource, onlyHelmify, onlyDeploy)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s", addon))
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
