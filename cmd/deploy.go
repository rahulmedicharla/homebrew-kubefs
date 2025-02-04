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
			cmd = exec.Command("sh", "-c", fmt.Sprintf("minikube status | grep Running"))
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Minikube not running: %v", err))
				return types.ERROR
			}

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
				defaultYaml = types.GetHelmChart(resource.DockerRepo, resource.Name, "ClusterIP", resource.Port, "false", "/health")
			}else{
				// frontend
				defaultYaml = types.GetHelmChart(resource.DockerRepo, resource.Name, "LoadBalancer", resource.Port, "true", "/")
			}

			fileWriteErr := os.WriteFile(fmt.Sprintf("%s/deploy/values.yaml", resource.Name), []byte(defaultYaml), 0644)

			if fileWriteErr != nil {
				return types.ERROR
			}

			err, valuesYaml := utils.ReadYaml(fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}
			env := valuesYaml["env"].([]interface{})
			for _, r := range utils.ManifestData.Resources {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
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

        utils.PrintWarning("Deploying all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := deployUnique(&resource, onlyHelmify, onlyDeploy)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s", resource.Name))
			}
        }

        utils.PrintSuccess("All resources deployed successfully")
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

		utils.PrintWarning(fmt.Sprintf("Deploying resource %v", names))

		for _, name := range names {
			var resource *types.Resource
			resource = utils.GetResourceFromName(name)

			if resource == nil {
				utils.PrintError(fmt.Sprintf("Resource %s not found", name))
				return
			}

			err := deployUnique(resource, onlyHelmify, onlyDeploy)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s", name))
				break
			}

		}

		utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", names))
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployAllCmd)
	deployCmd.AddCommand(deployResourceCmd)

	deployCmd.PersistentFlags().BoolP("only-helmify", "w", false, "only helmify the resources")
	deployCmd.PersistentFlags().BoolP("only-deploy", "d", false, "only deploy the resources")

}
